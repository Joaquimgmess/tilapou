package sim

import "testing"

func TestFeederKeepsTheTankStocked(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	run := func(withFeeder bool) (Micrograms, Coins) {
		s := stockedFarm(t, 21)
		s.Tanks[0].FeedStock = 5 * MicrogramsPerKilogram
		s.Cash = 5_000_000
		if withFeeder {
			s.grant(AutoFeeder)
		}

		out, err := Advance(Input{State: s, Until: 20 * TicksPerDay, Balance: b})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}

		return out.State.Tanks[0].Batches[0].MeanMass, out.State.Cash
	}

	manualMass, manualCash := run(false)
	autoMass, autoCash := run(true)

	if autoMass <= manualMass {
		t.Errorf("com comedouro o peixe pesou %d ug e sem comedouro %d: a automacao nao alimentou", autoMass, manualMass)
	}
	if autoCash >= manualCash {
		t.Errorf("comedouro comprou racao de graca: caixa auto=%d manual=%d", autoCash, manualCash)
	}
}

func TestAeratorAutomationTurnsOnWhenOxygenDrops(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 22)
	s.Tanks[0].Batches[0].Fish = 9_000
	s.Tanks[0].Batches[0].MeanMass = 600 * MicrogramsPerGram
	s.Tanks[0].Batches[0].MassRoot = massRootOf(600 * MicrogramsPerGram)
	s.Tanks[0].FeedStock = 50_000 * MicrogramsPerKilogram
	s.grant(AutoAerator)

	out, err := Advance(Input{State: s, Until: TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if !out.State.Tanks[0].Aerating && out.State.Tanks[0].Oxygen < b.Water.IdealMin {
		t.Error("o aerador automatico ficou desligado com oxigenio abaixo do ideal")
	}
}

func TestHarvesterSellsWhenTheBatchIsReady(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 23)
	s.Tanks[0].Batches[0].MeanMass = b.Growth.HarvestMass
	s.Tanks[0].Batches[0].MassRoot = massRootOf(b.Growth.HarvestMass)
	s.grant(AutoHarvester)

	before := s.Cash

	out, err := Advance(Input{State: s, Until: 10, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if out.State.Cash <= before {
		t.Error("o peao nao despescou o lote pronto")
	}
	if out.State.Tanks[0].Batches[0].Fish != 0 {
		t.Errorf("sobraram %d peixes apos a despesca automatica", out.State.Tanks[0].Batches[0].Fish)
	}
}

func TestContractPaysMoreForTheSameFish(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	run := func(withContract bool) Coins {
		s := stockedFarm(t, 24)
		s.Tanks[0].Batches[0].MeanMass = b.Growth.HarvestMass
		s.Cash = 0
		if withContract {
			s.grant(AutoContract)
		}

		actions := []Action{{ID: 1, Kind: ActionHarvest, At: 1, Tank: 1, Batch: 1}}

		out, err := Advance(Input{State: s, Until: 5, Balance: b, Actions: actions})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}

		return out.State.Cash
	}

	if with, without := run(true), run(false); with <= without {
		t.Errorf("contrato rendeu %d e venda avulsa %d", with, without)
	}
}

func TestPrestigeResetsTheFarmAndKeepsLifetime(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 25)
	s.LifetimeEarned = 100_000_000
	s.Cash = 999
	s.grant(AutoFeeder)

	out, err := Advance(Input{
		State:   s,
		Until:   5,
		Balance: b,
		Actions: []Action{{ID: 1, Kind: ActionPrestige, At: 1}},
	})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	got := out.State
	want := PrestigePointsFor(100_000_000, b.Progression.PrestigeDivisor)

	if got.Prestige != want {
		t.Errorf("prestigio = %d, want %d", got.Prestige, want)
	}
	if got.LifetimeEarned != 100_000_000 {
		t.Errorf("vitalicio = %d, prestigio nao pode zerar a base", got.LifetimeEarned)
	}
	if got.Cash != b.Progression.RestartCash {
		t.Errorf("caixa = %d, want %d", got.Cash, b.Progression.RestartCash)
	}
	if got.Upgrades != 0 {
		t.Error("prestigio manteve as automacoes compradas")
	}
	if got.TankCount != 1 || got.Tanks[0].Batches[0].Fish != b.Progression.RestartFish {
		t.Error("prestigio nao recomecou a fazenda com o lote inicial")
	}
}

func TestPrestigeIsRejectedWithoutEnoughLifetime(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 26)

	out, err := Advance(Input{
		State:   s,
		Until:   5,
		Balance: b,
		Actions: []Action{{ID: 1, Kind: ActionPrestige, At: 1}},
	})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if len(out.Outcomes) != 1 || out.Outcomes[0].Reason != RejectNotEnoughLifetime {
		t.Errorf("outcomes = %+v, queria rejeicao por vitalicio insuficiente", out.Outcomes)
	}
}

func TestPrestigeBonusSpeedsGrowth(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	run := func(points uint32) Micrograms {
		s := stockedFarm(t, 27)
		s.Prestige = points
		s.Tanks[0].FeedStock = 100_000 * MicrogramsPerKilogram

		out, err := Advance(Input{State: s, Until: 10 * TicksPerDay, Balance: b})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}

		return out.State.Tanks[0].Batches[0].MeanMass
	}

	if boosted, plain := run(20), run(0); boosted <= plain {
		t.Errorf("com prestigio o peixe pesou %d e sem prestigio %d", boosted, plain)
	}
}
