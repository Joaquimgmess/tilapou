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
	s.Tanks[0].Batches[0].MassRoot = massRootOf(380 * MicrogramsPerGram)
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
	if len(fish) != 12 || len(feed) != 12 {
		t.Fatalf("series com %d e %d pontos, queria 12", len(fish), len(feed))
	}
	if fish[11] != MarketAt(b, s.Tick).FishKg {
		t.Error("o ultimo ponto da serie nao e o preco de agora")
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

	id, ok := s.AddTank(TankEarthPond, b.Tanks[TankEarthPond].Litres)
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

	if fish, _ := s.StockAdvice(b, id); fish != 0 {
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
	s.LoadFeed(s.Tanks[0].ID, 5_000*MicrogramsPerKilogram)

	batch := s.Tanks[0].Batches[0]
	out := s.Forecast(b, s.Tanks[0].ID, batch.ID, batch.MeanMass+50*MicrogramsPerGram)

	if out.FeedEaten <= 0 {
		t.Fatal("a previsao nao diz quanta racao o lote come")
	}

	price := MarketAt(b, s.Tick).FeedKg
	eaten := Coins(mulDivFloor(int64(out.FeedEaten), int64(price), int64(MicrogramsPerKilogram)))

	if out.Cost >= eaten {
		t.Errorf("comendo do estoque ja pago, o gasto do periodo (%d) tinha que ficar abaixo do valor da racao comida (%d)",
			out.Cost, eaten)
	}
	if out.Cost <= 0 {
		t.Error("manutencao e energia deveriam aparecer no gasto mesmo sem comprar racao")
	}
}
