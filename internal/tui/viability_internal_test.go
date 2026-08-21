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

// Com prestigio a colher, a tela nao pode oferecer o recomeco: as duas portas reconstroem a
// mesma fazenda e tilapar devolve mais pontos, entao o [b] ali e a jogada dominada — e ela e
// irreversivel. A regra e "nunca as duas".
func TestComPrestigioAColherATelaOfereceTilaparENaoRecomecar(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 120, 40
	m.snapshot.Broke, m.snapshot.Prestige, m.snapshot.PrestigeNow = true, 3, 7

	rodape := strings.Join(m.keyHints(), " ")
	if strings.Contains(rodape, "b recomecar") {
		t.Errorf("o rodape oferece o recomeco com prestigio a colher: %q", rodape)
	}
	if !strings.Contains(rodape, "p tilapar") {
		t.Errorf("o rodape nao oferece tilapar com prestigio a colher: %q", rodape)
	}

	objetivo, ok := broke(m.snapshot)
	if !ok {
		t.Fatal("com a fazenda quebrada o objetivo nao apareceu")
	}
	if strings.Contains(objetivo.text, "[b]") {
		t.Errorf("o objetivo aponta o recomeco com prestigio a colher: %q", objetivo.text)
	}
	if !strings.Contains(objetivo.text, "[p]") {
		t.Errorf("o objetivo nao aponta tilapar: %q", objetivo.text)
	}
	// O verbo do [b] tem de estar na frase do [p], senao o jogador le tilapar como algo
	// guardado para depois e aperta o que ele entende.
	if !strings.Contains(objetivo.text, "recomeca do zero") {
		t.Errorf("o objetivo nao diz que tilapar recomeca do zero: %q", objetivo.text)
	}
	if !strings.Contains(objetivo.text, "4 matrizes") {
		t.Errorf("o objetivo nao diz quantas matrizes o jogador ganha: %q", objetivo.text)
	}

	// E o par: sem prestigio a colher, o recomeco continua sendo a saida oferecida.
	m.snapshot.PrestigeNow = 3
	if rodape := strings.Join(m.keyHints(), " "); !strings.Contains(rodape, "b recomecar") {
		t.Errorf("sem prestigio a colher o rodape deixou de oferecer o recomeco: %q", rodape)
	}
	if semPrestigio, _ := broke(m.snapshot); !strings.Contains(semPrestigio.text, "[b]") {
		t.Errorf("sem prestigio a colher o objetivo deixou de apontar o recomeco: %q", semPrestigio.text)
	}
}
