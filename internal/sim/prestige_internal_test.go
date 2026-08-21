package sim

import "testing"

func brokeFarm(t *testing.T, b *Balance) State {
	t.Helper()

	s := NewState(1, 0, 0)
	s.Debt = b.Credit.MaxPrincipal
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor - 1)
	s.Cash = 0

	if _, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
		t.Fatal("sem tanque")
	}

	return s
}

func TestABrokeFarmCanStartOverAndKeepsItsLifetime(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := brokeFarm(t, b)

	if !s.Broke(b, plansOf(t, b)) {
		t.Fatal("o cenario deveria estar quebrado: sem peixe, sem caixa, sem credito e sem prestigio")
	}

	for _, kind := range []ActionKind{ActionStock, ActionBorrow, ActionPrestige} {
		out, err := Advance(Input{State: s, Until: s.Tick + 2, Balance: b, Actions: []Action{
			{ID: 1, Kind: kind, Tank: 1, Amount: 1, At: s.Tick + 1},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcomes[0].Applied {
			t.Errorf("%v foi aplicada num estado que deveria estar sem saida", kind)
		}
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 2, Balance: b, Actions: []Action{
		{ID: 9, Kind: ActionRestart, At: s.Tick + 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Outcomes[0].Applied {
		t.Fatalf("recomecar foi recusado com %v: o beco nao tem saida", out.Outcomes[0].Reason)
	}

	after := out.State
	if after.Debt != 0 {
		t.Errorf("a divida sobreviveu ao recomeco: %d", after.Debt)
	}
	if after.Fish() == 0 || after.Cash <= 0 {
		t.Errorf("o recomeco nao devolveu peixe e caixa: %d peixes, %d de caixa", after.Fish(), after.Cash)
	}
	if after.LifetimeEarned != s.LifetimeEarned {
		t.Errorf("o faturamento vitalicio foi zerado: %d, queria %d", after.LifetimeEarned, s.LifetimeEarned)
	}
	if after.Prestige != s.Prestige {
		t.Errorf("recomecar quebrado nao pode dar prestigio: %d", after.Prestige)
	}

	lote := after.Tanks[0].Batches[0]
	if lote.Cost <= 0 {
		t.Error("o lote devolvido pelo recomeco nasceu de graca: a margem vai mostrar a venda inteira como lucro")
	}
	if after.Tanks[0].FeedUnitCost <= 0 {
		t.Error("a racao devolvida pelo recomeco nasceu de graca")
	}
}

func TestAHealthyFarmCannotStartOver(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 64)
	s.Cash = 100_000

	if s.Broke(b, plansOf(t, b)) {
		t.Fatal("uma fazenda com peixe e caixa nao esta quebrada")
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 2, Balance: b, Actions: []Action{
		{ID: 1, Kind: ActionRestart, At: s.Tick + 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Outcomes[0].Applied {
		t.Error("recomecar virou um reset de graca para quem nao quebrou")
	}
	if out.Outcomes[0].Reason != RejectNotBroke {
		t.Errorf("recusa saiu como %v, queria not_broke", out.Outcomes[0].Reason)
	}
}

func TestTilaparZeraADividaEDaPrestigio(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Debt, s.DebtCarry = 1_500_000, 77
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor) * 400

	if reason := prestige(&s, b, 10, &eventSink{}); reason != RejectNone {
		t.Fatalf("tilapada recusada: %v", reason)
	}

	if s.Debt != 0 || s.DebtCarry != 0 {
		t.Errorf("a tilapada deixou divida de %d com carry %d: o juro come o caixa de reinicio no mesmo tick",
			s.Debt, s.DebtCarry)
	}
	if s.Prestige == 0 {
		t.Error("a tilapada nao deu prestigio")
	}
}

// Limite livre so e saida quando o galpao solta o dinheiro: com o tanque vazio e caixa que
// nao povoa o piso do ciclo, o emprestimo e recusado, e ai a fazenda esta quebrada mesmo com
// o limite intacto. Contar o limite bruto deixava o jogador com toda tecla negando.
func TestOuOGalpaoSoltaOCreditoOuAFazendaEstaQuebrada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := brokeFarm(t, b)
	s.Debt = 0

	plans := plansOf(t, b)
	offer := s.LoanAdvice(b, s.Tanks[0].ID, plans[s.Tanks[0].Kind])

	if offer.Block != LoanOpen && !s.Broke(b, plans) {
		t.Fatalf("o galpao recusou (%v) e a fazenda nao contou como quebrada: nao sobra jogada nenhuma",
			offer.Block)
	}
	if offer.Block == LoanOpen && s.Broke(b, plans) {
		t.Fatalf("o galpao empresta %d e mesmo assim a fazenda conta como quebrada", offer.Cents)
	}
}

func TestFazendaSemPeixeSemCaixaESemCreditoQuebraMesmoComPrestigioASacar(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := brokeFarm(t, b)
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor) * 10

	if PrestigePointsFor(s.LifetimeEarned, b.Progression.PrestigeDivisor) <= s.Prestige {
		t.Fatal("o cenario precisa ter prestigio a sacar")
	}
	if !s.Broke(b, plansOf(t, b)) {
		t.Error("a fazenda tem 0 peixe, 0 caixa e 0 credito e nao esta quebrada: prestigio pendente nao e liquidez")
	}
}

func TestDividaAcimaDoTetoQuebraAFazendaSozinha(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 0
	s.Debt = b.Credit.BankruptcyPrincipal + 1
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor) * 400

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 900, 450*MicrogramsPerGram, 300_000)

	before := s.Prestige

	out, err := Advance(Input{State: s, Until: s.Tick + 1, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	if out.State.Debt != 0 {
		t.Errorf("a divida passou do teto e nao foi perdoada: %d", out.State.Debt)
	}
	if out.State.Cash != b.Progression.RestartCash {
		t.Errorf("a falencia deixou o caixa em %d, queria o pacote de reinicio %d",
			out.State.Cash, b.Progression.RestartCash)
	}
	if out.State.Prestige != before {
		t.Errorf("a falencia mexeu no prestigio: %d viroou %d", before, out.State.Prestige)
	}
	if out.State.LifetimeEarned != s.LifetimeEarned {
		t.Error("a falencia apagou o lifetime, que nunca desce")
	}

	var fell bool
	for _, e := range out.Events {
		if e.Kind == EventBankrupt {
			fell = true
		}
	}
	if !fell {
		t.Error("a falencia nao emitiu evento: o jogador nao fica sabendo por que a fazenda mudou")
	}
}

func TestDividaAbaixoDoTetoNaoQuebraNada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 500_000
	s.Debt = b.Credit.BankruptcyPrincipal - 1

	out, err := Advance(Input{State: s, Until: s.Tick + 1, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	if out.State.Debt == 0 {
		t.Error("a divida abaixo do teto foi perdoada")
	}
}

func TestFazendaSemAcaoPossivelQuebraEmTresDias(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 2_282
	// Limite consumido: credito que sobra e jogada na mao, e conta como caixa.
	s.Debt = b.Credit.MaxPrincipal

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	// Tres peixes minusculos: nao valem despesca e nao ha caixa para racao.
	s.StockTank(id, 3, 30*MicrogramsPerGram, 100)

	out, err := Advance(Input{State: s, Until: s.Tick + 4*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	if out.State.Debt != 0 {
		t.Errorf("depois de 4 dias sem acao possivel a divida ainda e %d", out.State.Debt)
	}
	// O caixa fica um pouco abaixo do pacote porque a manutencao do dia corre depois.
	if out.State.Cash <= 0 || out.State.Cash > b.Progression.RestartCash {
		t.Errorf("a falencia deixou o caixa em %d, esperava perto do pacote de reinicio %d",
			out.State.Cash, b.Progression.RestartCash)
	}
	if out.State.Fish() != b.Progression.RestartFish {
		t.Errorf("a falencia devolveu %d peixes, queria %d", out.State.Fish(), b.Progression.RestartFish)
	}
}

func TestFazendaComCaixaParaRacaoNaoQuebra(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 5_000_000
	s.Debt = 27_610

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 3, 30*MicrogramsPerGram, 100)

	out, err := Advance(Input{State: s, Until: s.Tick + 10*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	if out.State.Debt == 0 {
		t.Error("a fazenda com caixa para racao foi declarada falida")
	}
}

func TestBrokeOlhaOCustoDePovoarOMinimo(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := brokeFarm(t, b)
	// Alguns milhares de centavos parados: da para um alevino, nao da para povoar.
	s.Cash = 2_282

	if !s.Broke(b, plansOf(t, b)) {
		t.Error("com caixa que nao povoa o minimo a fazenda nao esta quebrada, e o [b] fica recusado")
	}
}

// Caixa alto de emprestimo com tanque vazio nao e uma fazenda viva: sem peixe, sem racao e
// sem caixa para um ciclo, nenhuma tecla faz nada. Contar isso pelo saco de racao dava ao
// jogador semanas de tela parada antes do resgate chegar.
func TestOCaixaPresoEmDividaContaComoFazendaTravada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	var plans Plans
	plans[TankEarthPond] = plan

	// A divida entra antes da conta: o juro dela corre durante o ciclo e faz parte do que o
	// caixa precisa cobrir, e o que sobra do limite conta junto com o caixa.
	s.Debt = b.Credit.MaxPrincipal

	tank := s.tank(id)
	cycle := s.cheapestCycle(b, tank, plan)

	s.Cash = cycle - 1
	if !s.stuck(b, plans) {
		t.Errorf("caixa %d nao paga o ciclo mais barato (%d) e a fazenda ainda conta como viva",
			s.Cash, cycle)
	}

	s.Cash = cycle
	if s.stuck(b, plans) {
		t.Errorf("caixa %d paga o ciclo mais barato e a fazenda conta como travada", s.Cash)
	}
}

// Racao no tanque so e saida enquanto houver lote comendo: com o tanque vazio ela e caixa que
// virou estoque parado, e tratar isso como fazenda viva desliga o resgate justamente para
// quem gastou o ultimo centavo se preparando para um ciclo que nao comecou.
func TestRacaoSemLoteNaoContaComoSaida(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.LoadFeed(id, 2_700*MicrogramsPerKilogram, 0)

	plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	var plans Plans
	plans[TankEarthPond] = plan

	s.Cash, s.Debt = 0, 1_199_700

	if !s.stuck(b, plans) {
		t.Error("tanque sem um peixe, com racao parada e sem caixa, ainda conta como fazenda viva")
	}
}

// Credito disponivel e jogada do jogador: contar so o caixa liga o cronometro de falencia em
// quem tem um emprestimo trivial na mao, e nega a tecla de recomecar a quem nao tem.
func TestOCreditoQueSobraContaDosDoisLados(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	novo := func(debt Coins) State {
		s := NewState(1, 0, 0)
		s.Debt = debt

		if _, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
			t.Fatal("sem tanque")
		}

		return s
	}

	plan := b.CycleAt(TankEarthPond, 0, 0)

	var plans Plans
	plans[TankEarthPond] = plan

	// Sem divida o limite inteiro esta na mao, mas quem decide e o que o galpao solta: as
	// duas contas tem de dar a mesma resposta, e a mesma que o conselho de credito da.
	folgado := novo(0)
	if folgado.stuck(b, plans) != folgado.Broke(b, plans) {
		t.Errorf("stuck = %v e Broke = %v na mesma fazenda: a tecla e o cronometro respondem perguntas diferentes",
			folgado.stuck(b, plans), folgado.Broke(b, plans))
	}
	if offer := folgado.LoanAdvice(b, folgado.Tanks[0].ID, plan); offer.Block == LoanOpen && folgado.stuck(b, plans) {
		t.Errorf("o galpao empresta %d e a fazenda conta como travada", offer.Cents)
	}

	// Com o limite quase todo consumido, o que sobra nao paga um ciclo: as duas concordam.
	apertado := novo(b.Credit.MaxPrincipal - 8_000)
	if !apertado.stuck(b, plans) {
		t.Error("o credito que sobra nao paga um ciclo e a fazenda ainda conta como viva")
	}
	if !apertado.Broke(b, plans) {
		t.Error("o objetivo manda recomecar e a tecla recusa: Broke discorda de stuck")
	}
}

// plansOf monta o plano do viveiro para as chamadas que precisam dele.
func plansOf(t *testing.T, b *Balance) Plans {
	t.Helper()

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, 0, 0)

	return plans
}
