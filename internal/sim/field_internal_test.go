package sim

import "testing"

func isothermalBalance(t *testing.T, celsius int32) *Balance {
	t.Helper()

	b := testBalance(t)
	b.Water.BaseTemp = MilliCelsius(celsius) * 1000
	b.Water.DailyTempSwing = 0
	b.Water.BiomassDrawPPM = 0
	b.Water.DailySwing = 0
	b.Water.BaselineOxygen = 7_000

	return b
}

func raise(t *testing.T, b *Balance, from Micrograms, days int64, fish FishCount) Micrograms {
	t.Helper()

	s := NewState(1, 0, 0)
	id, _ := s.addTank(TankEarthPond, 1_000_000)
	tank := s.tank(id)
	tank.addBatch(1, fish, from, 0)
	tank.FeedStock = 10_000_000 * MicrogramsPerKilogram

	out, err := Advance(Input{State: s, Until: Tick(days) * TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	return out.State.Tanks[0].Batches[0].MeanMass
}

func TestGrowthMatchesFieldData(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		celsius  int32
		from     Micrograms
		days     int64
		wantMinG int64
		wantMaxG int64
	}{
		"140 dias a 22C rende pouco": {22, 1 * MicrogramsPerGram, 140, 20, 90},
		"140 dias a 30C engorda":     {30, 1 * MicrogramsPerGram, 140, 300, 480},
		"abaixo de 18C nao cresce":   {16, 30 * MicrogramsPerGram, 60, 30, 34},
		"calor extremo penaliza":     {33, 30 * MicrogramsPerGram, 140, 60, 260},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := raise(t, isothermalBalance(t, tt.celsius), tt.from, tt.days, 1_000).Grams()
			if got < tt.wantMinG || got > tt.wantMaxG {
				t.Errorf("peso apos %d dias a %dC = %d g, queria entre %d e %d",
					tt.days, tt.celsius, got, tt.wantMinG, tt.wantMaxG)
			}
		})
	}
}

func TestDailyGainMatchesMeasuredRanges(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)

	tests := map[string]struct {
		from             Micrograms
		wantMin, wantMax int64
	}{
		"juvenil de 30g":  {30 * MicrogramsPerGram, 1, 3},
		"engorda de 100g": {100 * MicrogramsPerGram, 2, 4},
		"engorda de 400g": {400 * MicrogramsPerGram, 3, 7},
		"perto do abate":  {800 * MicrogramsPerGram, 1, 7},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gain := (raise(t, b, tt.from, 10, 1_000) - tt.from).Grams() / 10
			if gain < tt.wantMin || gain > tt.wantMax {
				t.Errorf("ganho diario partindo de %d g = %d g/dia, queria entre %d e %d",
					tt.from.Grams(), gain, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestHypoxiaOnlyKillsWhenDensityIsHigh(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fish      FishCount
		wantDeath bool
	}{
		"densidade folgada": {2_000, false},
		"superlotado":       {12_000, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			b := testBalance(t)
			s := stockedFarm(t, 4)
			s.Tanks[0].Batches[0].Fish = tt.fish
			s.Tanks[0].Batches[0].MeanMass = 600 * MicrogramsPerGram
			s.Tanks[0].Batches[0].MassRoot = massRootOf(600 * MicrogramsPerGram)
			s.Tanks[0].FeedStock = 100_000 * MicrogramsPerKilogram

			out, err := Advance(Input{State: s, Until: TicksPerDay, Balance: b})
			if err != nil {
				t.Fatalf("Advance() error = %v", err)
			}

			died := out.State.Tanks[0].Batches[0].Fish < tt.fish
			if died != tt.wantDeath {
				t.Errorf("morreu = %v com %d peixes, queria %v (sobraram %d)",
					died, tt.fish, tt.wantDeath, out.State.Tanks[0].Batches[0].Fish)
			}
		})
	}
}

func TestAeratorSavesTheOvercrowdedTank(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	run := func(aerating bool) FishCount {
		s := stockedFarm(t, 4)
		s.Tanks[0].Batches[0].Fish = 9_000
		s.Tanks[0].Batches[0].MeanMass = 600 * MicrogramsPerGram
		s.Tanks[0].Batches[0].MassRoot = massRootOf(600 * MicrogramsPerGram)
		s.Tanks[0].FeedStock = 100_000 * MicrogramsPerKilogram
		s.Tanks[0].Aerating = aerating

		out, err := Advance(Input{State: s, Until: TicksPerDay, Balance: b})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}

		return out.State.Tanks[0].Batches[0].Fish
	}

	without, with := run(false), run(true)
	if with <= without {
		t.Errorf("com aerador sobraram %d peixes, sem aerador %d: o aerador tem que salvar", with, without)
	}
}
