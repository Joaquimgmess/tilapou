package sim

import "testing"

// runToHarvest devolve o caixa que a fazenda ganha no ciclo de days dias: e o caixa, e nao a
// margem do lote, que o break-even promete zerar.
func runToHarvest(t *testing.T, b *Balance, fish, days int64) Coins {
	t.Helper()

	s := NewState(1, 0, 0)
	s.Cash = 100_000_000
	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	// Depois do tanque: a compra dele e custo de entrada, e nao do ciclo que roda dentro.
	start := s.Cash

	fingerlings := Coins(int64(b.Economy.FingerlingPrice) * fish)
	s.StockTank(id, FishCount(fish), b.Growth.FingerlingMass, fingerlings)
	s.Cash -= fingerlings
	s.SeedOxygen(b)
	tank := s.tank(id)
	tank.grant(AutoFeeder)
	tank.grant(AutoAerator)

	for day := range days {
		out, err := Advance(Input{State: s, Until: Tick(day+1) * TicksPerDay, Balance: b})
		if err != nil {
			t.Fatal(err)
		}
		s = out.State

		if s.Tanks[0].BatchCount == 0 {
			t.Fatalf("lote perdido com %d peixes", fish)
		}
	}

	batch := s.Tanks[0].Batches[0]
	price := b.PriceFor(batch.MeanMass, s.Tick)
	value := mulDivFloor(int64(price), int64(batch.Biomass()), int64(MicrogramsPerKilogram))
	left := mulDivFloor(int64(MarketAt(b, s.Tick).FeedKg),
		int64(s.Tanks[0].FeedStock), int64(MicrogramsPerKilogram))

	return s.Cash - start + Coins(value+left)
}

func TestBreakEvenStockLandsWithinTenFishOfTheCrossing(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	plan := b.CycleAt(TankEarthPond, 0, 0)

	if plan.BreakEven <= 0 {
		t.Fatal("sem ponto de equilibrio")
	}

	estimate := int64(plan.BreakEven)

	const slack = 10

	if cash := runToHarvest(t, b, estimate-slack, plan.Days); cash >= 0 {
		t.Errorf("%d peixes, %d abaixo da estimativa, ja fechou no azul com %d centavos",
			estimate-slack, slack, cash)
	}
	if cash := runToHarvest(t, b, estimate+slack, plan.Days); cash <= 0 {
		t.Errorf("%d peixes, %d acima da estimativa, ainda fechou no vermelho com %d centavos",
			estimate+slack, slack, cash)
	}
}

// A sonda roda com uma semente so, entao o break-even mostrado ao jogador nao pode mudar
// quando ela muda: se mudar, o numero e a sorte daquela temporada e nao a economia do tanque.
func TestOBreakEvenNaoDependeDaSementeDaSonda(t *testing.T) {
	t.Parallel()

	// O testBalance nao checa doenca, e sem surto qualquer semente da o mesmo numero: o
	// calendario de checagem entra aqui para a assercao ter o que cobrar.
	b := testBalance(t)
	b.Shock.CheckEvery = 5 * TicksPerDay

	want := b.cycleAtSeed(TankEarthPond, 0, 0, 1)

	for _, seed := range []Seed{2, 3} {
		if got := b.cycleAtSeed(TankEarthPond, 0, 0, seed); got != want {
			t.Errorf("semente %d deu %+v, e a semente 1 deu %+v", seed, got, want)
		}
	}
}
