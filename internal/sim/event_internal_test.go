package sim

import "testing"

func TestEventFishOnlyCountsFish(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	b.Shock.CheckEvery = 1
	for i := range b.Shock.DiseaseCount {
		b.Shock.Diseases[i].OutbreakPPM = UnitPPM
	}

	var day Tick
	for day = range Tick(365) {
		if _, ok := diseaseFor(b, seasonalTemp(b, day*TicksPerDay, 0)); ok {
			break
		}
	}

	s := NewState(1, 0, day*TicksPerDay)
	s.Cash = 1 << 40
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor) * 400

	id, ok := s.AddTank(TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 500, 200*MicrogramsPerGram, 0)
	s.SeedOxygen(b)

	out, err := Advance(Input{State: s, Until: s.Tick + 3, Balance: b, Actions: []Action{
		{ID: 1, Kind: ActionBuyUpgrade, Tank: id, Auto: AutoTechnician, At: s.Tick + 1},
		{ID: 2, Kind: ActionPrestige, At: s.Tick + 2},
	}})
	if err != nil {
		t.Fatal(err)
	}

	counts := map[EventKind]bool{
		EventHypoxiaDeaths: true, EventStarvationDeaths: true, EventDiseaseDeaths: true,
		EventHarvest: true, EventStocked: true, EventGrowth: true,
	}

	seen := map[EventKind]bool{}
	for _, e := range out.Events {
		seen[e.Kind] = true
		if !counts[e.Kind] && e.Fish != 0 {
			t.Errorf("evento %s carrega Fish=%d, mas esse campo e contagem de peixe", e.Kind, e.Fish)
		}
	}

	for _, kind := range []EventKind{EventUpgradeBought, EventPrestiged, EventDisease} {
		if !seen[kind] {
			t.Errorf("o cenario nao emitiu %s, entao esse sitio ficou sem cobertura", kind)
		}
	}
}
