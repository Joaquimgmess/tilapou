package tui

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

var plainFrame = regexp.MustCompile("\x1b\\[[0-9;]*m")

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
	d.run(d.model.(Model).Init())

	for step := range strings.SplitSeq(os.Getenv("QA_SCRIPT"), ",") {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		if days, ok := strings.CutPrefix(step, "d"); ok {
			n, err := strconv.Atoi(days)
			if err != nil {
				t.Fatalf("passo %q: %v", step, err)
			}
			qaJumpDays(t, n)
			d.press("r")

			continue
		}

		if step == "show" {
			fmt.Fprintf(os.Stdout, "\n%s\n", plainFrame.ReplaceAllString(d.model.(Model).render(), ""))

			continue
		}

		d.model, _ = d.model.Update(qaKey(step))
		d.run(nil)
		d.press("r")
	}

	fmt.Fprintf(os.Stdout, "\n%s\n", plainFrame.ReplaceAllString(d.model.(Model).render(), ""))
}
