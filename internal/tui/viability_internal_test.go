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

// O objetivo de divida nao pode mandar recomecar num estado em que o [b] responde que a
// fazenda nao quebrou: ele guardava so divida e caixa, nunca olhava o Broke nem o credito, e
// o @qa mediu isso disparando com 0,07 TC de divida e 4831,99 TC de credito aberto na tela.
func TestOObjetivoDeDividaNaoApontaTeclaRecusada(t *testing.T) {
	t.Parallel()

	comCredito := api.Snapshot{
		Debt: 7, CashCents: 0, Fish: 0, Broke: false,
		Tanks: []api.Tank{{ID: 1, LoanBlock: api.LoanOpen, LoanAdvice: 483_199}},
	}

	got, ok := crushingDebt(comCredito)
	if !ok {
		t.Fatal("com divida e caixa zero o objetivo nao apareceu")
	}
	if strings.Contains(got.text, "[b]") {
		t.Errorf("o objetivo aponta o recomeco com a fazenda de pe e credito aberto: %q", got.text)
	}
	if !strings.Contains(got.text, "[g]") {
		t.Errorf("o objetivo nao aponta o credito, que e a saida deste estado: %q", got.text)
	}

	// E o par: com a fazenda de fato quebrada, quem fala e o objetivo da quebra, e nao este.
	quebrada := comCredito
	quebrada.Broke = true
	if _, apareceu := crushingDebt(quebrada); apareceu {
		t.Error("com a fazenda quebrada o objetivo de divida ainda fala, e a tela passa a ter dois donos")
	}
}

// Na faixa em que falta racao, a tela precisa dizer que o credito paga: o jogador via "da
// para povoar, mas falta racao" e nao via o emprestimo que cobre a racao inteira.
func TestAFaixaDeRacaoCurtaCitaOCreditoQuandoEleExiste(t *testing.T) {
	t.Parallel()

	comCredito := api.Snapshot{
		CashCents: 20_000,
		Tanks: []api.Tank{{
			ID: 1, StockBlock: api.StockShortFeed, StockShort: 826_900,
			LoanBlock: api.LoanOpen, LoanAdvice: 1_033_845,
		}},
	}

	linha := emptyTankAdvice(comCredito, comCredito.Tanks[0])
	topo := farmGoal(comCredito)

	for nome, got := range map[string]string{"linha do tanque": linha, "objetivo": topo} {
		if !strings.Contains(got, "[g]") {
			t.Errorf("%s nao cita o credito que paga a racao: %q", nome, got)
		}
		if !strings.Contains(got, "[s]") {
			t.Errorf("%s deixou de citar a tecla que o jogo aceita: %q", nome, got)
		}
	}
	if !strings.Contains(topo, "8269,00") {
		t.Errorf("o objetivo nao diz quanto falta de racao: %q", topo)
	}

	// Sem credito, a frase fica como estava: apontar [g] fechado seria a mesma mentira de
	// antes, por outro lado.
	semCredito := comCredito
	semCredito.Tanks[0].LoanBlock, semCredito.Tanks[0].LoanAdvice = api.LoanNoCycle, 0
	if got := emptyTankAdvice(semCredito, semCredito.Tanks[0]); strings.Contains(got, "[g]") {
		t.Errorf("sem credito a linha aponta [g]: %q", got)
	}
}

// "venda X TC/kg" e receita REALIZADA por quilo, e nao o preco de mercado: ela ja divergia do
// mercado por preco de outros dias e por classe, e o bonus de contrato era o terceiro motivo.
// O rotulo e que mentia, entao ele muda; a conta fica.
func TestOUltimoCicloDizRecebidoENaoVenda(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 120, 40
	m.snapshot.LastCycle = api.Cycle{Fish: 2_000, MassGrams: 800_000, CostPerKg: 1_017, PricePerKg: 1_114, MarginCents: 21_336}

	frame := ansi.Strip(m.renderDashboard())
	if !strings.Contains(frame, "recebido") {
		t.Errorf("o painel do ultimo ciclo nao diz o que foi recebido:\n%s", frame)
	}
	if strings.Contains(frame, "venda 11,14") {
		t.Errorf("o painel ainda chama receita realizada de venda:\n%s", frame)
	}
}

// O contrato deixa de ser invisivel: ele muda o que o jogador recebe, e a linha nomeia quanto
// veio dele. Sem contrato, a linha nao existe — nao se gasta uma linha para dizer zero.
func TestADecisaoNomeiaOQueVeioDoContrato(t *testing.T) {
	t.Parallel()

	comContrato := api.Snapshot{
		Tanks: []api.Tank{{
			ID: 1, Fish: 2_000, Upgrades: []api.Upgrade{{Kind: "contrato", Owned: true}},
			Batches: []api.Batch{{
				ID: 1, Fish: 2_000, MeanGrams: 400, PriceKgCents: 717,
				Decision: api.Decision{SellNowCents: 26_887, ContractCents: 5_377},
			}},
		}},
	}

	got := contractLine(comContrato.Tanks[0].Batches[0])
	if !strings.Contains(got, "53,77") || !strings.Contains(got, "contrato") {
		t.Errorf("a linha nao nomeia o que veio do contrato: %q", got)
	}

	semContrato := comContrato.Tanks[0].Batches[0]
	semContrato.Decision.ContractCents = 0
	if got := contractLine(semContrato); got != "" {
		t.Errorf("sem contrato a linha existe e diz %q", got)
	}
}

// O aerador era um interruptor cego: [a] respondia "alternando" e nada na tela dizia se ele
// ficou ligado ou desligado — o @qa comparou os quadros antes e depois e as 32 linhas eram
// identicas. A coluna ESTADO passa a dizer, como ja diz do trato e da racao.
func TestOEstadoDizQuandoOAeradorEstaLigado(t *testing.T) {
	t.Parallel()

	tank := api.Tank{
		ID: 1, FeedKg: 180, OxygenUgL: 5_400, ServedFor: 240, Aerating: true,
	}
	batch := api.Batch{Fish: 2_000, Decision: api.Decision{DaysOfFeed: 63}}

	ligado, _ := rowState(&tank, &batch)
	if !strings.Contains(ligado, "aerando") {
		t.Errorf("com o aerador ligado o estado diz %q", ligado)
	}

	// E o par: desligado, a coluna volta a falar do que importa a seguir. Sem isso o teste
	// passaria com a coluna dizendo "aerando" para sempre.
	tank.Aerating = false
	desligado, _ := rowState(&tank, &batch)
	if strings.Contains(desligado, "aerando") {
		t.Errorf("com o aerador desligado o estado ainda diz %q", desligado)
	}

	// Com oxigenio critico e o aerador DESLIGADO, o alerta vence: e a hora de ligar.
	tank.Aerating, tank.OxygenUgL = false, 1_000
	if got, alert := rowState(&tank, &batch); !alert || !strings.Contains(got, "O2") {
		t.Errorf("com oxigenio critico e aerador desligado o estado diz %q (alerta=%v)", got, alert)
	}

	// Ligado, a mesma agua ruim mostra "aerando": o jogador precisa saber que ja esta agindo,
	// e nao levar um alerta do que ele acabou de resolver.
	tank.Aerating = true
	if got, _ := rowState(&tank, &batch); !strings.Contains(got, "aerando") {
		t.Errorf("com o aerador ligado contra oxigenio baixo o estado diz %q", got)
	}
}
