package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestTheGameBoyFrameIsTheSizeTheGuardClaims(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 200, 80

	frame := m.render()
	cols, rows := lipgloss.Width(frame), strings.Count(frame, "\n")+1

	if cols != gbCols || rows != gbRows {
		t.Errorf("o quadro mede %dx%d e a guarda promete %dx%d", cols, rows, gbCols, gbRows)
	}
}

func TestATerminalTooSmallForTheMapFallsBackToTheNumbers(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 80, 24

	frame := m.render()
	if !strings.Contains(frame, "DECISAO") {
		t.Errorf("sem espaco para o mapa deveria cair nos numeros:\n%s", frame)
	}
	if !strings.Contains(frame, "o mapa precisa de") {
		t.Errorf("a troca de tela precisa se explicar:\n%s", frame)
	}
}
