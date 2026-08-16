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
	tank := s.tank(id)
	tank.grant(AutoFeeder)
	tank.grant(AutoAerator)

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
		if now := Coins(value - int64(batch.Cost)); day == 0 || now > best {
			best = now
		}

		if batch.MeanMass >= topClass(b) {
			break
		}
	}

	return best
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

	if margin := runToHarvest(t, b, estimate-slack); margin >= 0 {
		t.Errorf("%d peixes, %d abaixo da estimativa, ja fechou no azul com %d centavos",
			estimate-slack, slack, margin)
	}
	if margin := runToHarvest(t, b, estimate+slack); margin <= 0 {
		t.Errorf("%d peixes, %d acima da estimativa, ainda fechou no vermelho com %d centavos",
			estimate+slack, slack, margin)
	}
}
