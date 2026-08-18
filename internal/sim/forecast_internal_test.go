package sim

import "testing"

func TestForecastReachesTheTargetAndPricesIt(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)
	s := stockedFarm(t, 61)
	s.Tanks[0].Batches[0].Fish = 500
	s.Tanks[0].FeedStock = 100_000 * MicrogramsPerKilogram
	s.Tanks[0].ServedUntil = Tick(maxInt32)
	s.Cash = 10_000_000

	got := s.Forecast(b, 1, 1, 600*MicrogramsPerGram)

	if !got.Reached {
		t.Fatalf("nao chegou em 600 g em %d dias (peso final %d g)", got.Days, got.MeanMass.Grams())
	}
	if got.Days <= 0 || got.Days > 300 {
		t.Errorf("dias = %d, fora do razoavel", got.Days)
	}
	if got.MeanMass < 600*MicrogramsPerGram {
		t.Errorf("peso final = %d g, queria pelo menos 600", got.MeanMass.Grams())
	}
	if got.Cost <= 0 {
		t.Error("engordar ate 600 g nao custou racao nenhuma")
	}
}

func TestForecastDoesNotTouchTheOriginalState(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)
	s := stockedFarm(t, 62)
	s.Tanks[0].ServedUntil = Tick(maxInt32)
	before := s

	s.Forecast(b, 1, 1, 900*MicrogramsPerGram)

	if s != before {
		t.Error("Forecast mexeu no estado recebido")
	}
}

func TestHoldingToTheNextClassBeatsSellingNowWhenFeedIsCheap(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)
	s := stockedFarm(t, 63)
	s.Tanks[0].Batches[0].Fish = 500
	s.Tanks[0].Batches[0].MeanMass = 380 * MicrogramsPerGram
	s.Tanks[0].Batches[0].MassRoot = MassRootOf(380 * MicrogramsPerGram)
	s.Tanks[0].FeedStock = 100_000 * MicrogramsPerKilogram
	s.Tanks[0].ServedUntil = Tick(maxInt32)
	s.Cash = 10_000_000

	now := s.Forecast(b, 1, 1, 380*MicrogramsPerGram)
	later := s.Forecast(b, 1, 1, 600*MicrogramsPerGram)

	if later.Value <= now.Value {
		t.Errorf("segurar ate a classe seguinte nao aumentou o valor: agora=%d depois=%d", now.Value, later.Value)
	}
}

func TestSeriesEndsAtTheCurrentTick(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 64)
	s.Tick = 50 * TicksPerDay

	fish, feed := s.Series(b, 12, TicksPerDay)

	wantFish := []Coins{1040, 1055, 1070, 1085, 1080, 1076, 1071, 1067, 1062, 1058, 1053, 1049}
	wantFeed := []Coins{330, 328, 327, 325, 324, 323, 322, 321, 320, 319, 318, 317}

	if !slicesEqual(fish, wantFish) {
		t.Errorf("serie do peixe = %v, queria %v", fish, wantFish)
	}
	if !slicesEqual(feed, wantFeed) {
		t.Errorf("serie da racao = %v, queria %v", feed, wantFeed)
	}
}

func TestForecastAssumesTheFarmKeepsBeingManaged(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)
	s := stockedFarm(t, 65)
	s.Tanks[0].Batches[0].Fish = 500
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].ServedUntil = 0

	got := s.Forecast(b, 1, 1, 600*MicrogramsPerGram)

	if !got.Reached {
		t.Fatalf("com tanque sem racao a previsao deixou o lote morrer: %d peixes de %d g em %d dias",
			got.Fish, got.MeanMass.Grams(), got.Days)
	}
	if got.Fish < 450 {
		t.Errorf("sobraram %d de 500 peixes na previsao: ela nao esta assumindo manejo", got.Fish)
	}
	if got.Cost <= 0 {
		t.Error("a previsao alimentou de graca: a racao precisa entrar no custo")
	}
}

func TestStockAdviceRefusesATankWithNoRoomForAnotherBatch(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 10_000_000

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	for range MaxBatchesPerTank {
		s.StockTank(id, 10, b.Growth.FingerlingMass, 0)
	}

	tank := s.tank(id)
	if room := tank.Capacity(b) - int64(tank.Fish()); room <= 0 {
		t.Fatalf("o cenario precisa de espaco de densidade sobrando, sobrou %d", room)
	}

	if fish, _ := s.StockAdvice(b, id, b.CycleAt(TankEarthPond, s.Tick, s.Zone)); fish != 0 {
		t.Errorf("com %d lotes o tanque nao aceita povoar, mas a sugestao foi de %d alevinos",
			tank.BatchCount, fish)
	}
}

func TestForecastReportsFeedEatenApartFromCost(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 500)
	s.Cash = 10_000_000
	s.Tanks[0].Aerating = true
	s.LoadFeed(s.Tanks[0].ID, 5_000*MicrogramsPerKilogram, MarketAt(b, s.Tick).FeedKg)

	batch := s.Tanks[0].Batches[0]
	out := s.Forecast(b, s.Tanks[0].ID, batch.ID, batch.MeanMass+50*MicrogramsPerGram)

	if out.FeedEaten <= 0 {
		t.Fatal("a previsao nao diz quanta racao o lote come")
	}

	price := MarketAt(b, s.Tick).FeedKg
	eaten := Coins(mulDivFloor(int64(out.FeedEaten), int64(price), int64(MicrogramsPerKilogram)))

	if out.Cost <= eaten {
		t.Errorf("o gasto do periodo (%d) tem que cobrir a racao comida (%d) mais manutencao e energia",
			out.Cost, eaten)
	}
}

func TestLoanAdviceSaysWhyItOffersNothing(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	for _, tc := range []struct {
		name  string
		setup func(*State) TankID
		block LoanBlock
	}{
		{"lotes cheios nao aceitam financiamento", func(s *State) TankID {
			id, _ := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
			for range MaxBatchesPerTank {
				s.StockTank(id, 10, b.Growth.FingerlingMass, 0)
			}

			return id
		}, LoanNoRoom},
		{"tanque no limite de densidade nao aceita financiamento", func(s *State) TankID {
			id, _ := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
			s.StockTank(id, 5_000, b.Growth.FingerlingMass, 0)

			return id
		}, LoanNoRoom},
		{"caixa que ja cobre o alvo dispensa emprestimo", func(s *State) TankID {
			id, _ := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
			s.StockTank(id, 100, b.Growth.FingerlingMass, 0)

			return id
		}, LoanNoNeed},
		{"credito estourado", func(s *State) TankID {
			s.Debt = b.Credit.MaxPrincipal
			id, _ := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)

			return id
		}, LoanNoCredit},
	} {
		s := NewState(1, 0, 0)
		s.Cash = 1 << 40
		id := tc.setup(&s)

		loan, block := s.LoanAdvice(b, id, CyclePlan{BreakEven: 968})
		if block != tc.block {
			t.Errorf("%s: motivo %v, queria %v", tc.name, block, tc.block)
		}
		if loan != 0 {
			t.Errorf("%s: ofereceu %d centavos mesmo bloqueado", tc.name, loan)
		}
	}
}

func TestPrevisaoNaoDependeDosOutrosTanques(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	target := Micrograms(600) * MicrogramsPerGram

	build := func(tanks int) State {
		s := NewState(1, 0, 0)
		s.Cash = 60_000
		for range tanks {
			id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
			if !ok {
				t.Fatal("sem tanque")
			}
			s.StockTank(id, 900, 450*MicrogramsPerGram, 300_000)
			s.LoadFeed(id, 20*MicrogramsPerKilogram, MarketAt(b, 0).FeedKg)
		}
		s.SeedOxygen(b)

		return s
	}

	one := build(1)
	want := one.Forecast(b, 1, 1, target)

	for _, tanks := range []int{4, 16} {
		many := build(tanks)
		if got := many.Forecast(b, 1, 1, target); got != want {
			t.Errorf("com %d tanques a previsao do lote 1 mudou:\n  got  %+v\n  want %+v", tanks, got, want)
		}
	}
}

func TestAPrevisaoSimulaSoOTanqueAlvo(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 10_000_000

	for range 8 {
		id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}
		s.StockTank(id, 900, 450*MicrogramsPerGram, 0)
	}

	only := s.isolate(5)

	if only.TankCount != 1 {
		t.Errorf("a previsao levou %d tanques para o Advance: o custo cresce com a fazenda inteira", only.TankCount)
	}
	if only.Tanks[0].ID != 5 {
		t.Errorf("isolou o tanque %d em vez do alvo 5", only.Tanks[0].ID)
	}
	if only.batch(5, only.Tanks[0].Batches[0].ID) == nil {
		t.Error("o lote alvo nao e mais alcancavel depois de isolar")
	}
}

func slicesEqual(got, want []Coins) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func TestPovoarDeixaCaixaParaOCustoFixoDoCiclo(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Cash = 2_000_000
	s.Debt = b.Credit.MaxPrincipal

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)
	if plan.Days <= 0 {
		t.Fatal("o ciclo do viveiro nao tem plano")
	}

	fish, perFish := s.StockAdvice(b, id, plan)
	if fish <= 0 {
		t.Fatal("o conselho nao sugeriu povoar nada")
	}

	spent := Coins(int64(fish)) * perFish
	left := s.Cash - spent

	fixed := Coins(plan.Days) * (b.Tanks[TankEarthPond].UpkeepPerDay +
		Coins(mulDivCeil(int64(s.Debt), int64(b.Credit.DailyRatePPM), int64(UnitPPM))))

	if left < fixed {
		t.Errorf("povoar %d peixes gasta %d e sobra %d, mas o ciclo de %d dias custa %d de juro e manutencao: o lote morre de fome antes de crescer",
			fish, spent, left, plan.Days, fixed)
	}
}
