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
	soUmaDirecao := map[ActionKind]bool{ActionBuyFeed: true}

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

// Recomecar e tilapar reconstroem a fazenda igual, mas tilapar devolve mais pontos. Com
// prestigio a colher, o [b] e a mesma porta com premio menor — e ele e irreversivel. O jogo
// recusa em vez de deixar o jogador queimar o que ja ganhou.
func TestRecomecarRecusaEnquantoDaParaTilapar(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 0
	s.Debt = b.Credit.MaxPrincipal
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor) * 400

	if _, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
		t.Fatal("sem tanque")
	}

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	if !s.Broke(b, plans) {
		t.Fatal("a fazenda deste teste precisa estar quebrada")
	}
	if PrestigePointsFor(s.LifetimeEarned, b.Progression.PrestigeDivisor) <= s.Prestige {
		t.Fatal("a fazenda deste teste precisa ter prestigio a colher")
	}

	// A copia sai antes: restart reconstroi a fazenda, e comparar depois seria comparar com o
	// resultado do proprio conserto.
	sem := s
	sem.LifetimeEarned = 0

	if reason := restart(&s, b, s.Tick, &eventSink{}, plans); reason == RejectNone {
		t.Error("o jogo aceitou recomecar com prestigio a colher: o jogador troca a mesma reconstrucao por menos pontos")
	}

	if reason := restart(&sem, b, sem.Tick, &eventSink{}, plans); reason != RejectNone {
		t.Errorf("sem prestigio a colher o jogo recusou recomecar: %v", reason)
	}
}

// Na faixa em que o jogo aceita povoar mas o caixa nao paga a racao, o conselho tem de dizer
// o numero que o jogo aceita. Devolver zero fazia a tela recusar a tecla sozinha, sem nunca
// falar com o dominio: as tres linhas mandavam [s], o [s] nao saia do cliente, e o estado
// nao mudava — sem saida por conta da propria tela.
func TestOConselhoDaONumeroQueOJogoAceitaMesmoSemRacaoPaga(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = Coins(mulDivCeil(int64(b.Economy.FingerlingPrice), MinStockFish, 1))

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	offer := s.StockAdvice(b, id, plan)
	if offer.Block != StockShortFeed {
		t.Fatalf("este teste precisa da faixa em que falta racao, e o bloco veio %v", offer.Block)
	}
	if offer.Fish <= 0 {
		t.Fatal("o conselho devolveu zero na faixa em que o jogo aceita povoar: a tela recusa a tecla sozinha")
	}

	reason, _ := applyStock(&s, b, Action{Kind: ActionStock, Tank: id, Amount: int64(offer.Fish)},
		s.Tick, &eventSink{})
	if reason != RejectNone {
		t.Errorf("o jogo recusou o numero que o conselho sugeriu: %v", reason)
	}
}

// Despescar e jogada quando a despesca levanta o piso da jogada mais barata: o criterio e
// VALOR, e nao massa. Com massa, a fazenda dizia "nao resta jogada possivel" e oferecia o
// recomeco irreversivel com 7553,00 TC de peixe no tanque.
func TestDespescarEJogadaPeloValorENaoPelaMassa(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	monta := func(fish FishCount, mass Micrograms) State {
		s := NewState(1, 0, 0)
		s.Cash = 0
		s.Debt = b.Credit.MaxPrincipal

		id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}
		s.StockTank(id, fish, mass, 1_000)

		return s
	}

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, 0, 0)

	// Lote verde, mas que vale muito mais que o piso: e saida, e das grandes.
	gordo := monta(2_000, 400*MicrogramsPerGram)
	if gordo.Broke(b, plans) {
		t.Errorf("fazenda com %d de peixe no tanque contou como quebrada: o [b] e irreversivel",
			gordo.harvestWorth(b))
	}

	// Tres alevinos minusculos continuam nao sendo saida: o piso e o mesmo da jogada mais
	// barata, e o lote magro nao chega la.
	magro := monta(3, 30*MicrogramsPerGram)
	if !magro.Broke(b, plans) {
		t.Errorf("fazenda com %d de peixe contou como viva: nao da nem para povoar o minimo",
			magro.harvestWorth(b))
	}
}
