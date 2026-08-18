package sim

import "testing"

func TestDeathOnTheOutbreakTickIsNotDecidedByTheOutbreakRoll(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	b.Shock.CheckEvery = 1

	for i := range b.Shock.DiseaseCount {
		b.Shock.Diseases[i].OutbreakPPM = UnitPPM / 10
	}

	var day Tick
	for day = range Tick(365) {
		if _, ok := diseaseFor(b, seasonalTemp(b, day*TicksPerDay, 0)); ok {
			break
		}
	}

	outbreaks, sameTickDeaths := 0, 0

	for seed := range 2_000 {
		s := NewState(Seed(seed+1), 0, day*TicksPerDay)
		s.Cash = 1 << 40

		id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}
		s.StockTank(id, 1_000, 200*MicrogramsPerGram, 0)
		s.SeedOxygen(b)
		s.tank(id).grant(AutoFeeder)

		out, err := Advance(Input{State: s, Until: s.Tick + 1, Balance: b})
		if err != nil {
			t.Fatal(err)
		}

		opened := false
		for _, e := range out.Events {
			if e.Kind == EventDisease {
				opened = true
			}
		}
		if !opened {
			continue
		}

		outbreaks++
		if out.State.Tanks[0].Accrual.DiseaseDeaths > 0 {
			sameTickDeaths++
		}
	}

	if outbreaks < 100 {
		t.Fatalf("so %d surtos em 2000 sementes: o cenario nao exercita a doenca", outbreaks)
	}

	rate := sameTickDeaths * 100 / outbreaks
	want := int(int64(b.Shock.Diseases[0].DeathPPM) * 100 / int64(UnitPPM))

	if rate > want+10 {
		t.Errorf("morreu peixe em %d%% dos ticks de surto, a taxa manda perto de %d%%: o sorteio da morte esta lendo a mesma chave do surto",
			rate, want)
	}
}
