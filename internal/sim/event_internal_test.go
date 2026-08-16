package sim

import "testing"

func TestEventFishOnlyCountsFish(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 64)
	s.Cash = 100_000_000

	out, err := Advance(Input{State: s, Until: s.Tick + TicksPerDay, Balance: b, Actions: []Action{
		{ID: 1, Kind: ActionBuyUpgrade, Tank: 1, Auto: AutoTechnician, At: s.Tick + 1},
	}})
	if err != nil {
		t.Fatal(err)
	}

	counts := map[EventKind]bool{
		EventHypoxiaDeaths: true, EventStarvationDeaths: true, EventDiseaseDeaths: true,
		EventHarvest: true, EventStocked: true, EventGrowth: true,
	}

	for _, e := range out.Events {
		if !counts[e.Kind] && e.Fish != 0 {
			t.Errorf("evento %s carrega Fish=%d, mas esse campo e contagem de peixe", e.Kind, e.Fish)
		}
	}
}
