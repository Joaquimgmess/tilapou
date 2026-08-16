package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	qaWidth  = 120
	qaHeight = 40
	keyEnter = "enter"
)

func qaJumpDays(t *testing.T, days int) {
	t.Helper()

	query := "UPDATE farms SET epoch = epoch - interval '" +
		strconv.Itoa(days*24*60) + " seconds'"

	cmd := exec.CommandContext(t.Context(), "docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "tilapou", "-d", os.Getenv("QA_DATABASE"), "-c", query)
	cmd.Dir = "../.."

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pulando %d dias: %v: %s", days, err, out)
	}
}

func qaKey(name string) tea.KeyPressMsg {
	switch name {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case keyEnter:
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}

	return tea.KeyPressMsg{Code: rune(name[0]), Text: name}
}

func TestQASession(t *testing.T) {
	t.Parallel()

	addr := os.Getenv("TILAPOU_DAEMON")
	if addr == "" {
		t.Skip("defina TILAPOU_DAEMON")
	}

	d := &driver{t: t, model: New(client.New(addr, 10*time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.run(d.model.(Model).fetch())

	qaPlay(t, d, os.Getenv("QA_SCRIPT"))

	fmt.Fprintf(os.Stdout, "\n%s\n", plain(d.model.(Model).render()))
}

func qaPlay(t *testing.T, d *driver, script string) {
	t.Helper()

	for step := range strings.SplitSeq(script, ",") {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		if days, ok := strings.CutPrefix(step, "d"); ok {
			if n, err := strconv.Atoi(days); err == nil {
				qaJumpDays(t, n)
				d.press("r")

				continue
			}
		}

		if step == "show" {
			fmt.Fprintf(os.Stdout, "\n%s\n", plain(d.model.(Model).render()))

			continue
		}

		var cmd tea.Cmd
		d.model, cmd = d.model.Update(qaKey(step))
		d.run(cmd)

		if !d.model.(Model).confirming {
			d.press("r")
		}
	}
}

func TestQAHarnessActuallySendsTheAction(t *testing.T) {
	t.Parallel()

	var posted []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := sizedSnapshot()
		if r.Method == http.MethodPost {
			var action client.Action
			_ = json.NewDecoder(r.Body).Decode(&action)
			posted = append(posted, action.Kind)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.run(d.model.(Model).fetch())

	qaPlay(t, d, "f,c,h")

	for _, want := range []string{"feed", "buy_feed", "harvest"} {
		if !slices.Contains(posted, want) {
			t.Errorf("o script mandou a tecla mas o daemon nao recebeu %q; recebeu %v", want, posted)
		}
	}
}

func TestQAScriptHandlesArrowsAndPrompts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		snap := sizedSnapshot()
		snap.PrestigeNow = 3
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.run(d.model.(Model).fetch())

	qaPlay(t, d, "down,up,left,right")

	qaPlay(t, d, "tab,p")
	if !d.model.(Model).confirming {
		t.Error("o refresh automatico cancelou o prompt antes do proximo passo do script")
	}
}
