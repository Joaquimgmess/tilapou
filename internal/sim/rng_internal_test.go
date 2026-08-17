package sim

import "testing"

func TestChanceSaturaNoTeto(t *testing.T) {
	t.Parallel()

	seed := Seed(20260817)
	key := RollKey{Purpose: PurposeMortality, Tick: 1, Tank: 1, Batch: 1}

	casos := map[PPM]bool{
		0:            false,
		-1:           false,
		UnitPPM:      true,
		UnitPPM + 1:  true,
		UnitPPM * 10: true,
	}

	for probability, want := range casos {
		if got := seed.Chance(key, probability); got != want {
			t.Errorf("Chance(%d) = %v, queria %v", probability, got, want)
		}
	}
}
