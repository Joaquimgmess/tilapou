package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestPainelUsaATelaInteira(t *testing.T) {
	t.Parallel()

	for _, size := range [][2]int{{120, 40}, {160, 50}, {200, 60}} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height, m.mode = size[0], size[1], ModeDashboard

		frame := m.render()
		if got := lipgloss.Height(frame); got != size[1] {
			t.Errorf("em %dx%d o painel desenhou %d linhas, sobrando %d mortas",
				size[0], size[1], got, size[1]-got)
		}

		for line := range strings.SplitSeq(frame, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Errorf("em %dx%d uma linha ficou com %d colunas", size[0], size[1], lipgloss.Width(line))

				break
			}
		}
	}
}

func TestDecisaoNaoQuebraAFraseComTelaLarga(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height, m.mode = 120, 40, ModeDashboard

	// A frase inteira tem de caber numa linha so: com largura fixa ela quebra entre o
	// "para pagar a" e o "manutencao", enquanto sobram colunas a direita.
	for line := range strings.SplitSeq(m.render(), "\n") {
		if strings.Contains(line, "minimo") && strings.Contains(line, "manutencao") {
			return
		}
	}

	t.Error("a linha de lotacao quebrou em duas mesmo com colunas livres a direita")
}
