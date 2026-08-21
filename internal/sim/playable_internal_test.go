package sim

import "testing"

// Fazenda com caixa que povoa nao esta quebrada: a tela mandava recomecar — que e
// irreversivel — com 4532,83 TC na barra e o [s] funcionando. stuck perguntava se cabia um
// ciclo inteiro, enquanto a acao perguntava se cabiam 100 alevinos.
func TestFazendaComCaixaQuePovoaNaoEstaQuebrada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 453_283
	s.Debt = 1_200_000

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	antes := s
	reason, _ := applyStock(&s, b, Action{Kind: ActionStock, Tank: id, Amount: MinStockFish},
		s.Tick, &eventSink{})
	if reason != RejectNone {
		t.Fatalf("o jogo recusou povoar o minimo com %d de caixa: %v", antes.Cash, reason)
	}

	if antes.Broke(b, plans) {
		t.Errorf("o jogo aceita povoar com %d de caixa e a tela manda recomecar do zero: [b] e irreversivel e queima o caixa",
			antes.Cash)
	}
}

// O contrario tem de valer tambem: sem caixa para a jogada mais barata e sem credito, a
// fazenda esta quebrada de verdade e o resgate precisa existir.
func TestFazendaSemCaixaParaAJogadaMaisBarataEstaQuebrada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 0
	s.Debt = b.Credit.MaxPrincipal

	if _, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
		t.Fatal("sem tanque")
	}

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	if !s.Broke(b, plans) {
		t.Error("sem caixa, sem credito e sem peixe a fazenda nao contou como quebrada: nao sobra jogada nenhuma")
	}
}

// O registro tem de responder por TODA acao: entrada vazia devolveria "nao e jogada" em
// silencio, que e como a divergencia entre a tela e o jogo nasceu quatro vezes.
func TestRegistroDeJogadasCobreTodaAcao(t *testing.T) {
	t.Parallel()

	if len(playable) != int(actionKindCount) {
		t.Fatalf("o registro tem %d entradas e existem %d acoes", len(playable), actionKindCount)
	}

	for kind, can := range playable {
		if can == nil {
			t.Errorf("a acao %v nao tem entrada no registro de jogadas", ActionKind(kind))
		}
	}
}

// A entrada do registro replica o piso do applyX correspondente, e nao um piso proprio: com
// o saco de 100 kg como piso, a fazenda com caixa para 1 kg contava como quebrada, a tela
// mandava recomecar — irreversivel — e comprar 3,27 TC de racao a trazia de volta.
func TestComprarRacaoContaComOQueOJogoAceita(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Debt = b.Credit.MaxPrincipal
	s.Cash = MarketAt(b, s.Tick).FeedKg

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 500, 100*MicrogramsPerGram, 1_000)
	s.tank(id).FeedStock = 0

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	antes := s
	reason, _ := applyBuyFeed(&s, b, Action{Kind: ActionBuyFeed, Tank: id, Amount: 1}, s.Tick, &eventSink{})
	if reason != RejectNone {
		t.Fatalf("o jogo recusou comprar 1 kg com %d de caixa: %v", antes.Cash, reason)
	}

	if antes.Broke(b, plans) {
		t.Errorf("o jogo aceita comprar racao com %d de caixa e a tela manda recomecar do zero", antes.Cash)
	}
}

// O piso de cada entrada do registro tem de ser o piso da propria acao, e o oraculo e o
// apply — nao um numero escrito aqui. A busca binaria mede os dois e exige o mesmo caixa: um
// teste que copiasse os pisos espelharia o conserto em vez de cobra-lo, e foi por isso que a
// suite ficou verde com o registro cobrando saco de 100 kg onde a acao aceita 1 kg.
//
// Duas entradas afrouxam por decisao, e a excecao mora aqui declarada: comprar racao so vale
// com lote para comer, e despescar so vale com lote no ponto. Nelas o contrato e de uma
// direcao — o registro nunca pode ser mais PERMISSIVO que a acao.
func TestOPisoDoRegistroEOPisoDaAcao(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	// A direcao que a suite deixava aberta e a cara: registro mais rigoroso que a acao marca
	// a fazenda como quebrada com jogada na mao, e o [b] e irreversivel.
	soUmaDirecao := map[ActionKind]bool{ActionBuyFeed: true, ActionHarvest: true}

	casos := map[ActionKind]func(id TankID) Action{
		ActionStock:   func(id TankID) Action { return Action{Kind: ActionStock, Tank: id, Amount: MinStockFish} },
		ActionBuyFeed: func(id TankID) Action { return Action{Kind: ActionBuyFeed, Tank: id, Amount: 1} },
		ActionTreat:   func(id TankID) Action { return Action{Kind: ActionTreat, Tank: id} },
		ActionBuyTank: func(TankID) Action {
			return Action{Kind: ActionBuyTank, TankKind: TankEarthPond}
		},
	}

	plan := b.CycleAt(TankEarthPond, 0, 0)

	for kind, montar := range casos {
		registro := menorCaixa(t, b, plan, func(s *State, tank *Tank, plan CyclePlan) bool {
			return playable[kind](s, b, tank, plan)
		})
		acao := menorCaixa(t, b, plan, func(s *State, _ *Tank, _ CyclePlan) bool {
			copia := *s
			var plans Plans
			plans[TankEarthPond] = plan
			reason, _ := dispatch(&copia, b, montar(copia.Tanks[0].ID), copia.Tick, &eventSink{}, plans)

			return reason == RejectNone
		})

		if registro == acao {
			continue
		}
		if soUmaDirecao[kind] && registro > acao {
			continue
		}

		t.Errorf("%v: o registro vira jogada com caixa %d e a acao so aceita com %d", kind, registro, acao)
	}
}

// menorCaixa acha, por busca binaria, o menor caixa em que a pergunta responde sim.
func menorCaixa(t *testing.T, b *Balance, plan CyclePlan, responde func(*State, *Tank, CyclePlan) bool) Coins {
	t.Helper()

	monta := func(cash Coins) (*State, *Tank, CyclePlan) {
		s := NewState(1, 0, 0)
		s.Cash = cash

		id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}
		s.StockTank(id, 200, b.Growth.HarvestMass, 1_000)
		s.tank(id).Batches[0].Sick = 100
		s.tank(id).FeedStock = 0

		return &s, s.tank(id), plan
	}

	baixo, alto := Coins(0), Coins(100_000_000)
	if s, tank, plan := monta(alto); !responde(s, tank, plan) {
		return alto
	}

	for baixo < alto {
		meio := baixo + (alto-baixo)/2
		if s, tank, plan := monta(meio); responde(s, tank, plan) {
			alto = meio
		} else {
			baixo = meio + 1
		}
	}

	return baixo
}
