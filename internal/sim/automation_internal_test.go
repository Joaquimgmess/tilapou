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
			s.Tanks[0].grant(AutoFeeder)
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
	s.Tanks[0].Batches[0].MassRoot = MassRootOf(600 * MicrogramsPerGram)
	s.Tanks[0].FeedStock = 50_000 * MicrogramsPerKilogram
	s.Tanks[0].grant(AutoAerator)

	out, err := Advance(Input{State: s, Until: TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if !out.State.Tanks[0].Aerating && out.State.Tanks[0].Oxygen < b.Water.AeratorOff {
		t.Error("o aerador automatico ficou desligado com oxigenio abaixo do ideal")
	}
}

func TestHarvesterSellsWhenTheBatchIsReady(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 23)
	s.Tanks[0].Batches[0].MeanMass = b.Growth.HarvestMass
	s.Tanks[0].Batches[0].MassRoot = MassRootOf(b.Growth.HarvestMass)
	s.Tanks[0].grant(AutoHarvester)

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
		s.Tanks[0].ServedUntil = Tick(maxInt32)
		s.Cash = 0
		if withContract {
			s.Tanks[0].grant(AutoContract)
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
	s.Tanks[0].grant(AutoFeeder)

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
	if got.Cash > b.Progression.RestartCash || got.Cash < b.Progression.RestartCash-100 {
		t.Errorf("caixa = %d, esperava perto de %d", got.Cash, b.Progression.RestartCash)
	}
	if got.Tanks[0].Upgrades != 0 {
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
		s.Tanks[0].ServedUntil = Tick(maxInt32)

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

func TestHarvestedBatchLeavesTheTank(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 31)
	s.Cash = 10_000_000

	actions := []Action{
		{ID: 1, Kind: ActionHarvest, At: 1, Tank: 1, Batch: 1},
		{ID: 2, Kind: ActionStock, At: 3, Tank: 1, Amount: 500},
	}

	out, err := Advance(Input{State: s, Until: 10, Balance: b, Actions: actions})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	tank := out.State.Tanks[0]
	if tank.BatchCount != 1 {
		t.Fatalf("tanque ficou com %d lotes, queria 1: o lote despescado nao saiu", tank.BatchCount)
	}
	if tank.Batches[0].Fish != 500 || tank.Batches[0].MeanMass > 2*b.Growth.FingerlingMass {
		t.Errorf("o lote que sobrou nao e o novo: %d peixes de %d g",
			tank.Batches[0].Fish, tank.Batches[0].MeanMass.Grams())
	}
}

func TestTankKeepsAcceptingBatchesAfterManyHarvests(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 32)
	s.Cash = 10_000_000

	var id ActionID

	actions := make([]Action, 0, 12)
	for cycle := range 6 {
		id++
		actions = append(actions, Action{ID: id, Kind: ActionHarvest, At: Tick(cycle*10 + 1), Tank: 1, Batch: BatchID(cycle + 1)})
		id++
		actions = append(actions, Action{ID: id, Kind: ActionStock, At: Tick(cycle*10 + 3), Tank: 1, Amount: 100})
	}

	out, err := Advance(Input{State: s, Until: 100, Balance: b, Actions: actions})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	for _, outcome := range out.Outcomes {
		if !outcome.Applied && outcome.Reason == RejectTankFull {
			t.Fatal("o tanque encheu de lotes vazios apos varias despescas")
		}
	}
}

func TestFeederBuysWhatItCanAfford(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 33)
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].grant(AutoFeeder)
	s.Cash = Coins(50 * int64(b.Market.FeedBasePerKg))

	out, err := Advance(Input{State: s, Until: 5, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if out.State.Tanks[0].FeedStock <= 0 {
		t.Error("o comedouro nao comprou nada mesmo com caixa para 50 kg")
	}
	if out.State.Cash < 0 {
		t.Errorf("o comedouro gastou mais do que tinha: caixa = %d", out.State.Cash)
	}
}

func TestFeederStopsWhenBroke(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 34)
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].grant(AutoFeeder)
	s.Cash = 1

	out, err := Advance(Input{State: s, Until: 5, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if out.State.Cash < 0 {
		t.Errorf("caixa ficou negativo em vez de virar divida: %d", out.State.Cash)
	}
	if out.State.Tanks[0].FeedStock != 0 {
		t.Error("comprou racao sem ter dinheiro")
	}
}

func TestOAeradorSeDesligaEmVezDeEndividar(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	for _, tc := range []struct {
		name    string
		cash    Coins
		wantOn  bool
		wantDue bool
	}{
		{"com caixa o aerador segue ligado e cobra", b.Economy.AeratorCostTick * 100, true, false},
		{"sem caixa ele se desliga", b.Economy.AeratorCostTick - 1, false, false},
	} {
		s := NewState(1, 0, 0)
		s.Cash = tc.cash

		id, ok := s.AddTank(TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}
		s.StockTank(id, 100, 200*MicrogramsPerGram, 0)
		s.SeedOxygen(b)
		s.tank(id).Aerating = true

		out, err := Advance(Input{State: s, Until: s.Tick + 1, Balance: b})
		if err != nil {
			t.Fatal(err)
		}

		if on := out.State.Tanks[0].Aerating; on != tc.wantOn {
			t.Errorf("%s: aerador ligado=%v, queria %v", tc.name, on, tc.wantOn)
		}
		if due := out.State.Debt > 0; due != tc.wantDue {
			t.Errorf("%s: divida=%d, o aerador nunca pode endividar", tc.name, out.State.Debt)
		}
	}
}
