package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

var (
	escapes    = regexp.MustCompile("\x1b\\[[0-9;]*m")
	errOffline = errors.New("connection refused")
)

func plain(frame string) string {
	return escapes.ReplaceAllString(frame, "")
}

func frameAfter(t *testing.T, msgs ...tea.Msg) (model Model, frame string) {
	t.Helper()

	m := New(nil)
	m.mode = ModeDashboard
	m.width, m.height = 100, 35

	var next tea.Model = m
	for _, msg := range msgs {
		next, _ = next.Update(msg)
	}
	updated, ok := next.(Model)
	if !ok {
		t.Fatal("Update devolveu outro tipo")
	}

	return updated, plain(updated.render())
}

func TestAFailedPollKeepsTheLastFrameAndMarksItsAge(t *testing.T) {
	t.Parallel()

	good := snapshotMsg{snapshot: sizedSnapshot()}
	bad := snapshotMsg{err: errOffline}

	_, frame := frameAfter(t, good, tickMsg{}, bad, tickMsg{}, bad)

	if !strings.Contains(frame, "DECISAO") {
		t.Errorf("o quadro sumiu depois de um poll perdido:\n%s", frame)
	}
	if !strings.Contains(frame, "sem resposta") {
		t.Errorf("o quadro nao avisa que esta velho:\n%s", frame)
	}
}

func TestTheFirstFailureStillShowsTheFullScreenError(t *testing.T) {
	t.Parallel()

	_, frame := frameAfter(t, snapshotMsg{err: errOffline})

	if !strings.Contains(frame, "daemon fora do ar") {
		t.Errorf("sem daemon e sem snapshot deveria explicar o problema:\n%s", frame)
	}
}

func TestARefusedActionStaysOnScreenLongEnoughToRead(t *testing.T) {
	t.Parallel()

	press := tea.KeyPressMsg{Code: 'p', Text: "p"}
	m, frame := frameAfter(t, snapshotMsg{snapshot: sizedSnapshot()}, press, tickMsg{}, tickMsg{})

	if !strings.Contains(frame, "Ainda nao da para tilapar") {
		t.Errorf("a recusa sumiu antes de dois ticks:\n%s", frame)
	}

	for range messageTicks {
		next, _ := m.Update(tickMsg{})
		aged, ok := next.(Model)
		if !ok {
			t.Fatal("Update devolveu outro tipo")
		}
		m = aged
	}
	if strings.Contains(m.render(), "Ainda nao da para tilapar") {
		t.Error("a mensagem nunca expira")
	}
}

func TestTheTankMenuIsVisibleInTheDashboard(t *testing.T) {
	t.Parallel()

	press := tea.KeyPressMsg{Code: 'z', Text: "z"}
	_, frame := frameAfter(t, snapshotMsg{snapshot: sizedSnapshot()}, press)

	if !strings.Contains(frame, "Servir o trato") {
		t.Errorf("o menu do tanque nao apareceu no painel:\n%s", frame)
	}
}

func TestEachSessionStartsFromADifferentIdempotencyKey(t *testing.T) {
	t.Parallel()

	seen := map[uint64]bool{}
	for range 100 {
		key := New(nil).nextKey
		if seen[key] {
			t.Fatalf("duas sessoes nasceram com a chave %d: a segunda vai ter as acoes engolidas", key)
		}
		seen[key] = true
	}
}

func TestTheFooterOnlyPromisesKeysThatWork(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()

	for _, mode := range []Mode{ModeDashboard, ModeGameBoy} {
		m := New(nil)
		m.snapshot, m.mode = snap, mode
		m.width, m.height = 120, 40
		m.menu = tankMenu(snap, snap.Tanks[0])

		footer := plain(m.render())
		footer = footer[strings.LastIndex(footer, "\n")+1:]

		for _, dead := range []string{"f trato", "c racao", "h despescar", "tab mapa", "q sair", "q sai"} {
			if strings.Contains(footer, dead) {
				t.Errorf("modo %d: com menu aberto o rodape promete %q, que nao faz nada: %q", mode, dead, footer)
			}
		}
		if !strings.Contains(footer, "z confirma") {
			t.Errorf("modo %d: o rodape do menu nao diz como confirmar: %q", mode, footer)
		}
	}
}

func TestJAndKMoveTheMenuCursor(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	m := New(nil)
	m.snapshot = snap
	m.menu = tankMenu(snap, snap.Tanks[0])

	next, _ := m.onMenuKey("j")
	moved, ok := next.(Model)
	if !ok {
		t.Fatal("onMenuKey devolveu outro tipo")
	}
	if moved.menu.cursor != 1 {
		t.Errorf("j deixou o cursor em %d, queria 1", moved.menu.cursor)
	}

	back, _ := moved.onMenuKey("k")
	if back.(Model).menu.cursor != 0 {
		t.Errorf("k nao voltou o cursor: %d", back.(Model).menu.cursor)
	}
}

func TestABlockedLoanSaysWhyItIsBlocked(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	tank := &snap.Tanks[0]
	tank.BreakEven, tank.StockAdvice, tank.LoanAdvice = int64(tank.Fish)+500, 0, 0

	item := creditItems(snap, *tank)[0]
	if item.enabled {
		t.Fatal("sem espaco no credito o item tinha que estar desabilitado")
	}
	if strings.Contains(item.hint, "cobre") {
		t.Errorf("o item esta bloqueado mas a dica explica para que ele serviria: %q", item.hint)
	}
	if !strings.Contains(item.hint, "credito") {
		t.Errorf("a dica nao diz que o limite de credito acabou: %q", item.hint)
	}
}

func TestEveryRejectReasonHasAMessage(t *testing.T) {
	t.Parallel()

	reasons := []string{
		"no_such_tank", "no_such_batch", "not_enough_cash", "not_enough_feed", "tank_full",
		"farm_full", "bad_amount", "too_dense", "already_owned", "not_enough_lifetime",
		"credit_limit", "no_debt", "nothing_sick", "not_broke", "unknown_kind",
	}

	for _, reason := range reasons {
		got := explain(&client.Outcome{Reason: reason}, 0)
		if strings.Contains(got, reason) {
			t.Errorf("o motivo %q chega cru na tela: %q", reason, got)
		}
	}
}
