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
			if lipgloss.Width(got) > max(tc.width, lipgloss.Width(tc.text)) {
				t.Errorf("clipTo(%q, %d) = %q, mais largo que %d colunas", tc.text, tc.width, got, tc.width)
			}
		})
	}
}
