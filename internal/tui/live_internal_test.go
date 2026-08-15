package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

type driver struct {
	t     *testing.T
	model tea.Model
}

func (d *driver) run(cmd tea.Cmd) {
	d.t.Helper()

	for range 8 {
		if cmd == nil {
			return
		}

		msg := cmd()
		switch batch := msg.(type) {
		case tea.BatchMsg:
			var next tea.Cmd
			for _, inner := range batch {
				if inner == nil {
					continue
				}
				sub := inner()
				if _, isTick := sub.(tickMsg); isTick {
					continue
				}
				d.model, next = d.model.Update(sub)
			}
			cmd = next
		case nil:
			return
		default:
			d.model, cmd = d.model.Update(msg)
		}
	}
}

func (d *driver) press(key string) {
	d.t.Helper()

	code := rune(0)
	if len(key) == 1 {
		code = rune(key[0])
	}

	msg := tea.KeyPressMsg{Code: code, Text: key}
	switch key {
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	}

	var cmd tea.Cmd
	d.model, cmd = d.model.Update(msg)
	d.run(cmd)
}

func (d *driver) frame(title string) string {
	d.t.Helper()

	return "\n===== " + title + " =====\n" + d.model.View().Content
}

func TestLiveSession(t *testing.T) {
	addr := os.Getenv("TILAPOU_DAEMON")
	if addr == "" {
		t.Skip("defina TILAPOU_DAEMON")
	}

	var model tea.Model = New(client.New(addr, 5*time.Second))
	d := &driver{t: t, model: model}
	d.run(model.Init())

	var out strings.Builder
	out.WriteString(d.frame("1. abertura"))

	for range 3 {
		d.press("up")
	}
	out.WriteString(d.frame("2. de frente para o viveiro"))

	d.press("z")
	out.WriteString(d.frame("3. menu do tanque"))

	d.press("z")
	out.WriteString(d.frame("4. servi o trato"))

	d.press("down")
	d.press("down")
	d.press("down")
	d.press("down")
	d.press("down")
	out.WriteString(d.frame("5. menu com cursor em outra opcao"))

	d.press("x")
	d.press("m")
	out.WriteString(d.frame("6. painel de numeros"))

	fmt.Fprint(os.Stdout, out.String())

	if err := os.WriteFile("/tmp/live.txt", []byte(out.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}
