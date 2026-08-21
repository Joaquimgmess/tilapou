package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Joaquimgmess/tilapou/internal/api"
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
		// Com as duas no vermelho, comparar perda por dia elegia a perda mais lenta.
		"os dois no vermelho, segurar perde mais": {
			hold: -131_760, sell: -104_019, holdDays: 104, cycleDays: 196, want: false,
		},
		"os dois no vermelho, segurar perde menos": {
			hold: -104_019, sell: -131_760, holdDays: 104, cycleDays: 196, want: true,
		},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := holdWins(api.Decision{
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
				before := api.Decision{HoldMargin: hold, SellNowMargin: sell, HoldDays: 10, CycleDays: 100}
				after := api.Decision{HoldMargin: hold - extra, SellNowMargin: sell, HoldDays: 10, CycleDays: 100}

				if !holdWins(before) && holdWins(after) {
					t.Fatalf("segurar %d custando %d a mais passou a ganhar de vender %d", hold, extra, sell)
				}
			}
		}
	}
}

func TestTanqueVazioNaoEntraEmAlerta(t *testing.T) {
	t.Parallel()

	tank := api.Tank{ID: 1, Fish: 0, FeedKg: 0, ServedFor: 0, OxygenUgL: 6_000}
	vazio := api.Batch{ID: 1, Fish: 0}

	label, alert := rowState(&tank, &vazio)
	if alert {
		t.Errorf("tanque sem um peixe entrou em alerta como %q", label)
	}
	if label != "vazio" {
		t.Errorf("tanque sem peixe diz %q, queria dizer que esta vazio", label)
	}
}

func TestTanqueVazioNaoAnunciaProximaClasse(t *testing.T) {
	t.Parallel()

	// Lote sem peixe nao tem proxima classe para anunciar.
	vazio := api.Batch{ID: 1, Fish: 0, MeanGrams: 0, NextClassGrams: 0}

	if got := nextClass(&vazio); got != "no topo" {
		t.Errorf("lote de 0 g anuncia %q", got)
	}
}

func TestLotacaoNaoDizNoAzulComOLoteNoVermelho(t *testing.T) {
	t.Parallel()

	tank := api.Tank{
		ID: 1, Fish: 718, Capacity: 5_000, BreakEven: 376,
		Batches: []api.Batch{{ID: 1, Fish: 718, MarginCents: -104_019}},
	}

	if got := ansi.Strip(renderStocking(tank)); strings.Contains(got, "no azul") {
		t.Errorf("a lotacao diz %q com o lote perdendo 1040,19 TC", got)
	}
}

func TestLotacaoDizNoAzulQuandoOLoteEstaNoAzul(t *testing.T) {
	t.Parallel()

	tank := api.Tank{
		ID: 1, Fish: 718, Capacity: 5_000, BreakEven: 376,
		Batches: []api.Batch{{ID: 1, Fish: 718, MarginCents: 21_336}},
	}

	if got := ansi.Strip(renderStocking(tank)); !strings.Contains(got, "no azul") {
		t.Errorf("a lotacao nao diz no azul com o lote ganhando: %q", got)
	}
}

// A barra de teclas nao pode passar da largura da tela: o que sobra e cortado, e o que sai
// primeiro e o fim da linha, onde mora a tecla de sair.
func TestABarraDeTeclasCabeNaTela(t *testing.T) {
	t.Parallel()

	for _, width := range []int{40, 60, 80, 100, 120, 200} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height = width, 40

		// Conselho apontando outro tanque: e o estado que soma a dica de pulo na barra.
		m.snapshot.Tanks = append(m.snapshot.Tanks, m.snapshot.Tanks[0])
		m.snapshot.Tanks[1].ID = 2
		m.snapshot.Tanks[1].Fish = 0
		m.snapshot.Broke = true
		m.snapshot.PrestigeNow = m.snapshot.Prestige + 1

		if got := lipgloss.Height(m.renderKeys()); got != 1 {
			t.Errorf("em %d colunas a barra ocupa %d linhas: ela nao coube e quebrou\n%s",
				width, got, ansi.Strip(m.renderKeys()))
		}
	}
}

// A tecla que tira o jogador do estado travado nao pode ser a unica que ele nao ve: hoje o
// jogo so a cita no texto de objetivo, que ele pode nao estar lendo.
func TestABarraAnunciaARecomecarQuandoAFazendaQuebra(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 120, 40

	if bar := ansi.Strip(m.renderKeys()); strings.Contains(bar, "b recomecar") {
		t.Errorf("a fazenda esta de pe e a barra ja oferece recomecar: %q", bar)
	}

	m.snapshot.Broke = true

	if bar := ansi.Strip(m.renderKeys()); !strings.Contains(bar, "b recomecar") {
		t.Errorf("a fazenda quebrou e a barra nao diz como recomecar: %q", bar)
	}
}

// Tres telas diferentes mandam povoar, e todas tem de sair do mesmo conselho que a tecla usa:
// mandar apertar o que o jogo vai recusar e pior que nao dizer nada.
func TestNenhumaTelaMandaPovoarQuandoOConselhoEZero(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 200, 80

	for i := range m.snapshot.Tanks {
		m.snapshot.Tanks[i].Fish = 0
		m.snapshot.Tanks[i].Batches = nil
		// O motivo acompanha o numero: conselho zero com bloco "open" e um estado que o daemon
		// nao produz, e testar por ele seria testar uma fazenda que nao existe.
		m.snapshot.Tanks[i].StockAdvice, m.snapshot.Tanks[i].StockBlock = 0, api.StockNoCash
	}

	telas := map[string]string{
		"painel de numeros":  ansi.Strip(m.renderDashboard()),
		"conselho do tanque": tankAdvice(m.snapshot, m.snapshot.Tanks[0]),
	}

	for name, got := range telas {
		if strings.Contains(got, "povoe com [s]") {
			t.Errorf("%s manda povoar com o conselho em zero", name)
		}
	}
}

// Em terminal pequeno o painel substitui o mapa, mas o aviso disso nao pode tomar a linha do
// conselho: quem joga em tela pequena fica sem orientacao nenhuma no estado critico.
func TestOAvisoDeTelaPequenaNaoTomaALinhaDoConselho(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 100, 30

	if m.fitsGameBoy() {
		t.Fatal("o cenario precisa de uma tela que nao cabe o mapa")
	}

	m.snapshot.Tanks[0].Fish = 0
	m.snapshot.Tanks[0].Batches = nil
	m.snapshot.Tanks[0].StockAdvice = 0

	goal, _ := objective(m.snapshot, m.tankID())
	if got := ansi.Strip(m.renderGoal()); !strings.Contains(got, clipTo(goal, m.effectiveWidth()-panelInset)) {
		t.Errorf("a linha do conselho diz %q em vez do objetivo %q", got, goal)
	}
}
