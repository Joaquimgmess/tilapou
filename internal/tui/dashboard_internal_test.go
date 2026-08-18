package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestClipToCortaPorColunaENaoPorByte(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		text  string
		width int
		want  string
	}{
		"cabe inteiro":        {text: "não deu", width: 20, want: "não deu"},
		"corta com acento":    {text: "não deu: ração", width: 8, want: "não deu~"},
		"largura minima":      {text: "não", width: 1, want: "não"},
		"acento na fronteira": {text: "ãããã", width: 3, want: "ãã~"},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := clipTo(tc.text, tc.width)
			if got != tc.want {
				t.Errorf("clipTo(%q, %d) = %q, queria %q", tc.text, tc.width, got, tc.want)
			}
			if tc.width > 1 && lipgloss.Width(got) > tc.width {
				t.Errorf("clipTo(%q, %d) = %q, mais largo que %d colunas", tc.text, tc.width, got, tc.width)
			}
		})
	}
}

func TestAheadOlhaParaOsQuatroLados(t *testing.T) {
	t.Parallel()

	a := avatar{x: 5, y: 5, facing: facingUp}

	casos := map[facing][2]int{
		facingUp:    {5, 4},
		facingDown:  {5, 6},
		facingLeft:  {4, 5},
		facingRight: {6, 5},
	}

	for look, want := range casos {
		a.facing = look

		x, y := a.ahead()
		if x != want[0] || y != want[1] {
			t.Errorf("ahead() olhando para %d = (%d,%d), queria (%d,%d)", look, x, y, want[0], want[1])
		}
	}
}
