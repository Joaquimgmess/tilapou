package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

func TestOMenuCabeNoTamanhoMinimo(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Tanks[0].Upgrades = everyUpgrade()

	menus := map[string]*menu{
		"tanque": tankMenu(snap, snap.Tanks[0]),
		"galpao": shedMenu(snap, snap.Tanks[0]),
	}

	for name, overlay := range menus {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, size := range [][2]int{{gbCols, gbRows}, {gbCols, gbRows + 3}, {100, 38}, {120, 40}} {
				m := New(nil)
				m.snapshot = snap
				m.width, m.height, m.mode = size[0], size[1], ModeGameBoy
				m.menu = overlay
				// A recusa entra acima do menu, e e nesse quadro que o topo sumia.
				m = m.say("Sem grana: custa 5247,03 TC e faltam 1702,19 TC")

				frame := m.render()
				if got := lipgloss.Height(frame); got > size[1] {
					t.Errorf("em %dx%d o quadro saiu com %d linhas: o topo do menu rola para fora da tela",
						size[0], size[1], got)
				}

				// O item sob o cursor sublinha por palavra e injeta ANSI entre elas, entao a
				// comparacao e sobre o texto limpo.
				plain := ansi.Strip(frame)
				for _, want := range []string{overlay.title, overlay.items[0].label, "┏"} {
					if !strings.Contains(plain, want) {
						t.Errorf("em %dx%d o menu perdeu %q:\n%s", size[0], size[1], want, plain)
					}
				}
			}
		})
	}
}
