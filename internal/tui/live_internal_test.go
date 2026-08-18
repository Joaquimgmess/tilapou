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

// refresh pede o snapshot pelo mesmo caminho do jogo, que registra o pedido em voo.
func (d *driver) refresh() {
	d.t.Helper()

	started, cmd := d.model.(Model).fetching()
	d.model = started
	d.run(cmd)
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
	case keyUp:
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case keyDown:
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case keyLeft:
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case keyRight:
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

// Pedido novo com o anterior no ar enfileira trabalho no daemon em vez de esperar por ele: e
// assim que a TUI se afoga sozinha quando a resposta demora mais que o intervalo de refresh.
func TestORefreshNaoPedeDeNovoComOPedidoNoAr(t *testing.T) {
	t.Parallel()

	m := New(nil)

	first, cmd := m.fetching()
	if cmd == nil {
		t.Fatal("o primeiro refresh nao pediu nada")
	}

	if _, again := first.fetching(); again != nil {
		t.Error("pediu de novo com o anterior ainda no ar")
	}

	// Resposta do pedido que a acao cancelou: chega depois e traz um mundo mais velho.
	if stale := first.onSnapshot(snapshotMsg{snapshot: sizedSnapshot(), seq: first.flightSeq - 1}); stale.inFlight == flightNone {
		t.Error("resposta de um pedido cancelado foi aceita e liberou o caminho")
	}

	back := first.onSnapshot(snapshotMsg{snapshot: sizedSnapshot(), seq: first.flightSeq})
	if _, after := back.fetching(); after == nil {
		t.Error("depois da resposta chegar o refresh parou de pedir")
	}
}

// Peixe nadando com o numero parado e a tela dizendo que esta viva quando o que ela mostra ja
// nao vale: passando do timeout, a animacao congela.
func TestAAnimacaoCongelaQuandoODadoEnvelhece(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 200, 80
	m.frame = 3

	fresh := m.render()

	m.staleTicks = staleFreeze
	if frozen := m.render(); frozen == fresh {
		t.Error("com o dado velho a tela continua animando igual")
	}

	m.frame = 4
	if next := m.render(); next != m.withFrame(3).render() {
		t.Error("com o dado velho a animacao andou de quadro")
	}
}

func (m Model) withFrame(frame int) Model {
	m.frame = frame

	return m
}

// Confirmacao destrutiva nao pode aceitar a tecla de uma acao comum: quem apertar povoar por
// reflexo diante de um prompt zera tanques, caixa e automacoes.
func TestAConfirmacaoDestrutivaNaoAceitaATeclaDePovoar(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.confirming, m.restarting = true, true

	after, cmd := m.onConfirm("s")
	if cmd != nil {
		t.Error("a tecla de povoar confirmou o recomecar")
	}

	if next, ok := after.(Model); ok && next.confirming {
		t.Error("o prompt continuou aberto depois da tecla")
	}
}
