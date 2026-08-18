package sim

import "testing"

func TestRollBelowFicaDentroDoIntervalo(t *testing.T) {
	t.Parallel()

	seed := Seed(20260817)

	for tick := range Tick(2000) {
		key := RollKey{Purpose: PurposeMortality, Tick: tick, Tank: 1, Batch: 1}

		got := seed.RollBelow(key, int64(UnitPPM))
		if got < 0 || got >= int64(UnitPPM) {
			t.Fatalf("RollBelow no tick %d saiu de [0, %d): %d", tick, UnitPPM, got)
		}
		if only := seed.RollBelow(key, 1); only != 0 {
			t.Fatalf("RollBelow com bound 1 no tick %d devolveu %d, queria 0", tick, only)
		}
	}
}

func TestChanceRespeitaOsExtremos(t *testing.T) {
	t.Parallel()

	seed := Seed(20260817)

	for tick := range Tick(2000) {
		key := RollKey{Purpose: PurposeMortality, Tick: tick, Tank: 1, Batch: 1}

		if seed.Chance(key, 0) {
			t.Fatalf("Chance(0) no tick %d saiu verdadeira", tick)
		}
		if !seed.Chance(key, UnitPPM) {
			t.Fatalf("Chance(UnitPPM) no tick %d saiu falsa", tick)
		}
	}
}
