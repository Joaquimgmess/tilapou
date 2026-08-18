package tui

import (
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
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

func TestQuemLevaOMarcadorDeMelhorNegocio(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		hold, sell          int64
		holdDays, cycleDays int64
		want                bool
	}{
		"segurar ganha por um centavo":           {hold: 1_001, sell: 1_000, want: true},
		"vender ganha por um centavo":            {hold: 1_000, sell: 1_001, want: false},
		"empate vai para vender agora":           {hold: 1_000, sell: 1_000, want: false},
		"segurar no vermelho, vender pior ainda": {hold: -100, sell: -500, want: true},
		"os dois no vermelho, vender menos pior": {hold: -500, sell: -100, want: false},
		// Numeros medidos no viveiro cheio: segurar de 600 g ate 900 g rende 294.747 a
		// mais, mas leva 150 dias (1.965/dia) contra 8.663/dia de recomecar o ciclo.
		"ganho grande, mas espalhado em meses": {
			hold: 1_897_445, sell: 1_602_698, holdDays: 150, cycleDays: 185, want: false,
		},
		"ganho pequeno, mas em poucos dias": {
			hold: 1_700_000, sell: 1_602_698, holdDays: 10, cycleDays: 185, want: true,
		},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := holdWins(client.Decision{
				HoldMargin: tc.hold, SellNowMargin: tc.sell,
				HoldDays: tc.holdDays, CycleDays: tc.cycleDays,
			})
			if got != tc.want {
				t.Errorf("holdWins(segurar %d em %d d, vender %d, ciclo %d d) = %v, queria %v",
					tc.hold, tc.holdDays, tc.sell, tc.cycleDays, got, tc.want)
			}
		})
	}
}

func TestCustoMaiorDeSegurarNuncaMelhoraOVeredito(t *testing.T) {
	t.Parallel()

	for hold := int64(-3); hold <= 3; hold++ {
		for sell := int64(-3); sell <= 3; sell++ {
			for extra := int64(1); extra <= 5; extra++ {
				before := client.Decision{HoldMargin: hold, SellNowMargin: sell, HoldDays: 10, CycleDays: 100}
				after := client.Decision{HoldMargin: hold - extra, SellNowMargin: sell, HoldDays: 10, CycleDays: 100}

				if !holdWins(before) && holdWins(after) {
					t.Fatalf("segurar %d custando %d a mais passou a ganhar de vender %d", hold, extra, sell)
				}
			}
		}
	}
}
