package sim

import "testing"

func brokeFarm(t *testing.T, b *Balance) State {
	t.Helper()

	s := NewState(1, 0, 0)
	s.Debt = b.Credit.MaxPrincipal
	s.LifetimeEarned = Coins(b.Progression.PrestigeDivisor - 1)
	s.Cash = 0

	if _, ok := s.AddTank(TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
		t.Fatal("sem tanque")
	}

	return s
}

func TestABrokeFarmCanStartOverAndKeepsItsLifetime(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := brokeFarm(t, b)

	if !s.Broke(b) {
		t.Fatal("o cenario deveria estar quebrado: sem peixe, sem caixa, sem credito e sem prestigio")
	}

	for _, kind := range []ActionKind{ActionStock, ActionBorrow, ActionPrestige} {
		out, err := Advance(Input{State: s, Until: s.Tick + 2, Balance: b, Actions: []Action{
			{ID: 1, Kind: kind, Tank: 1, Amount: 1, At: s.Tick + 1},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcomes[0].Applied {
			t.Errorf("%v foi aplicada num estado que deveria estar sem saida", kind)
		}
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 2, Balance: b, Actions: []Action{
		{ID: 9, Kind: ActionRestart, At: s.Tick + 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Outcomes[0].Applied {
		t.Fatalf("recomecar foi recusado com %v: o beco nao tem saida", out.Outcomes[0].Reason)
	}

	after := out.State
	if after.Debt != 0 {
		t.Errorf("a divida sobreviveu ao recomeco: %d", after.Debt)
	}
	if after.Fish() == 0 || after.Cash <= 0 {
		t.Errorf("o recomeco nao devolveu peixe e caixa: %d peixes, %d de caixa", after.Fish(), after.Cash)
	}
	if after.LifetimeEarned != s.LifetimeEarned {
		t.Errorf("o faturamento vitalicio foi zerado: %d, queria %d", after.LifetimeEarned, s.LifetimeEarned)
	}
	if after.Prestige != s.Prestige {
		t.Errorf("recomecar quebrado nao pode dar prestigio: %d", after.Prestige)
	}

	lote := after.Tanks[0].Batches[0]
	if lote.Cost <= 0 {
		t.Error("o lote devolvido pelo recomeco nasceu de graca: a margem vai mostrar a venda inteira como lucro")
	}
	if after.Tanks[0].FeedUnitCost <= 0 {
		t.Error("a racao devolvida pelo recomeco nasceu de graca")
	}
}

func TestAHealthyFarmCannotStartOver(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 64)
	s.Cash = 100_000

	if s.Broke(b) {
		t.Fatal("uma fazenda com peixe e caixa nao esta quebrada")
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 2, Balance: b, Actions: []Action{
		{ID: 1, Kind: ActionRestart, At: s.Tick + 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Outcomes[0].Applied {
		t.Error("recomecar virou um reset de graca para quem nao quebrou")
	}
	if out.Outcomes[0].Reason != RejectNotBroke {
		t.Errorf("recusa saiu como %v, queria not_broke", out.Outcomes[0].Reason)
	}
}
