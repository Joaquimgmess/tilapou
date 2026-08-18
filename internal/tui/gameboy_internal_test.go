package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestTheGameBoyFrameIsTheSizeTheGuardClaims(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 200, 80

	frame := m.render()
	rows := strings.Count(frame, "\n") + 1

	if rows != gbRows {
		t.Errorf("o quadro tem %d linhas e a guarda promete %d", rows, gbRows)
	}
	if cols := lipgloss.Width(frame); cols != m.width {
		t.Errorf("o quadro tem %d colunas numa tela de %d: sobra fundo do terminal em volta", cols, m.width)
	}

	// O aparelho fica centrado dentro do quadro, e e ele que a guarda dimensiona.
	device := 0
	for line := range strings.SplitSeq(ansi.Strip(frame), "\n") {
		device = max(device, lipgloss.Width(strings.TrimSpace(line)))
	}

	if device != gbCols {
		t.Errorf("o aparelho desenhado mede %d colunas e a guarda promete %d", device, gbCols)
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
