package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
