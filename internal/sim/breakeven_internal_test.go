package sim

import "testing"

func runToHarvest(t *testing.T, b *Balance, fish int64) Coins {
	t.Helper()

	s := NewState(1, 0, 0)
	s.Cash = 100_000_000
	id, ok := s.AddTank(TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, FishCount(fish), b.Growth.FingerlingMass, Coins(int64(b.Economy.FingerlingPrice)*fish))
	s.SeedOxygen(b)
	s.Tanks[0].Upgrades = 1<<AutoFeeder | 1<<AutoAerator

	var best Coins

	for day := range int64(breakEvenCapDays) {
		out, err := Advance(Input{State: s, Until: Tick(day+1) * TicksPerDay, Balance: b})
		if err != nil {
			t.Fatal(err)
		}
		s = out.State

		if s.Tanks[0].BatchCount == 0 {
			t.Fatalf("lote perdido com %d peixes", fish)
		}
		batch := s.Tanks[0].Batches[0]
		price := b.PriceFor(batch.MeanMass, s.Tick)
		value := mulDivFloor(int64(price), int64(batch.Biomass()), int64(MicrogramsPerKilogram))
		best = max(best, Coins(value-int64(batch.Cost)))

		if batch.MeanMass >= topClass(b) {
			break
		}
	}

	return best
}

func TestBreakEvenStockMatchesASimulatedCycle(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	plan := b.CycleAt(TankEarthPond, 0, 0)

	if plan.BreakEven <= 0 {
		t.Fatal("sem ponto de equilibrio")
	}

	if margin := runToHarvest(t, b, int64(plan.BreakEven)); margin < 0 {
		t.Errorf("povoar o equilibrio (%d peixes) fechou em %d centavos", plan.BreakEven, margin)
	}
	if margin := runToHarvest(t, b, int64(plan.BreakEven)/2); margin > 0 {
		t.Errorf("metade do equilibrio (%d peixes) fechou em %d centavos, deveria dar prejuizo",
			plan.BreakEven/2, margin)
	}
}
