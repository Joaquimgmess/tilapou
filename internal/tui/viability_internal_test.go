package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// O painel do mercado fala de PRECO do dia; resultado do lote e do painel ao lado. Dizer "da
// lucro" ali punha os dois a discordar no mesmo quadro: a equivalencia dizia lucro enquanto a
// DECISAO dizia ABAIXO DO CUSTO e o lote perdia dinheiro. As duas afirmacoes eram verdadeiras
// e mediam coisas diferentes — o que estava errado era o rotulo.
func TestOMercadoNaoDaVeredictoSobreOLote(t *testing.T) {
	t.Parallel()

	for _, ratio := range []int64{2_320_000, 1_100_000} {
		got := ansi.Strip(viability(ratio, 1_250_000))

		for _, avoid := range []string{"lucro", "inviavel"} {
			if strings.Contains(got, avoid) {
				t.Errorf("com equivalencia %d o painel do mercado da veredicto: %q", ratio, got)
			}
		}
		// O piso ja chega no cliente e hoje e gasto so para produzir um adjetivo: mostra-lo
		// ensina o limiar e tira a leitura da dependencia da cor.
		if !strings.Contains(got, "1,25") {
			t.Errorf("com equivalencia %d o painel nao mostra o piso: %q", ratio, got)
		}
	}
}

// A DECISAO continua dando o veredicto do lote: e ela que tem o custo do peixe.
func TestADecisaoContinuaDizendoQuandoOLotePerde(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 120, 40
	m.snapshot.Tanks[0].Batches[0].Decision.BreakEvenPerKg = 900
	m.snapshot.Tanks[0].Batches[0].PriceKgCents = 678

	if frame := ansi.Strip(m.renderDashboard()); !strings.Contains(frame, "ABAIXO DO CUSTO") {
		t.Errorf("com o preco abaixo do custo a DECISAO deixou de avisar:\n%s", frame)
	}
}

// Com caixa que povoa mas nao alimenta, a tela nao pode apontar o recomeco: o [b] responde
// not_broke e o [s] funciona. O aviso e sobre a racao, e a tecla citada e a que age.
func TestComCaixaQuePovoaMasNaoAlimentaATelaNaoMandaRecomecar(t *testing.T) {
	t.Parallel()

	snap := api.Snapshot{
		CashCents: 20_000,
		Tanks: []api.Tank{{
			ID: 1, StockBlock: api.StockShortFeed, StockShort: 826_900,
			LoanBlock: api.LoanNoCycle,
		}},
	}

	linha := emptyTankAdvice(snap, snap.Tanks[0])
	topo := farmGoal(snap)

	for nome, got := range map[string]string{"linha do tanque": linha, "objetivo do topo": topo} {
		if strings.Contains(got, "[b]") {
			t.Errorf("%s aponta o recomeco num estado em que o [s] funciona: %q", nome, got)
		}
		if !strings.Contains(got, "[s]") {
			t.Errorf("%s nao aponta a tecla que age: %q", nome, got)
		}
		if !strings.Contains(got, "racao") {
			t.Errorf("%s nao avisa que a racao nao esta paga: %q", nome, got)
		}
	}
}

// A fazenda quebrada diz o que e verdade — que nao resta jogada — em vez de enumerar motivos
// que a propria tela desmente duas linhas abaixo.
func TestAFazendaQuebradaNaoEnumeraMotivos(t *testing.T) {
	t.Parallel()

	quebrada := api.Snapshot{Broke: true, Debt: 700_000, CashCents: 0, Fish: 500}

	got, ok := broke(quebrada)
	if !ok {
		t.Fatal("com a fazenda quebrada o objetivo nao apareceu")
	}
	if strings.Contains(got.text, "sem peixe") {
		t.Errorf("a frase afirma 'sem peixe' com 500 peixes na fazenda: %q", got.text)
	}
	if !strings.Contains(got.text, "jogada") {
		t.Errorf("a frase nao diz o que e verdade — que nao resta jogada: %q", got.text)
	}
}
