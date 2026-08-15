package sim

import "testing"

func TestDeathOnTheOutbreakTickIsNotDecidedByTheOutbreakRoll(t *testing.T) {
	t.Parallel()

	const (
		fish     = FishCount(1_000)
		risk     = PPM(90_000)
		deathPPM = 41
		rounds   = 4_000
	)

	outbreaks, deaths := 0, 0
	for i := range rounds {
		seed := Seed(i + 1)
		if !seed.Chance(RollKey{Tick: 7, Tank: 1, Batch: 1, Purpose: PurposeDisease}, risk) {
			continue
		}
		outbreaks++

		batch := Batch{ID: 1, Fish: fish, MeanMass: MicrogramsPerGram, Sick: 3}
		if killFish(&batch, deathPPM, seed, RollKey{Tick: 7, Tank: 1, Batch: 1, Purpose: PurposeDiseaseDeath}) > 0 {
			deaths++
		}
	}

	if outbreaks == 0 {
		t.Fatal("nenhum surto no cenario")
	}

	got := deaths * 100 / outbreaks
	want := int(int64(fish) * deathPPM / int64(UnitPPM) * 100)
	if got > want+10 {
		t.Errorf("morte extra em %d%% dos surtos, esperado perto de %d%%: o sorteio da morte esta reusando o do surto",
			got, want+4)
	}
}
