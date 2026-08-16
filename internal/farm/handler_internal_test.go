package farm

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func viewOfStocked(t *testing.T, b *sim.Balance) (TankView, SnapshotView) {
	t.Helper()

	s := sim.NewState(1, 0, 0)
	s.Cash = 5_000_000

	id, ok := s.AddTank(sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 900, 450*sim.MicrogramsPerGram, 300_000)
	s.LoadFeed(id, 200*sim.MicrogramsPerKilogram, sim.MarketAt(b, 0).FeedKg)
	s.SeedOxygen(b)

	view := viewOf(Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}, b, newPlans())
	if len(view.Tanks) == 0 {
		t.Fatal("a view saiu sem tanque")
	}

	return view.Tanks[0], view
}

func brokeView(t *testing.T, b *sim.Balance) SnapshotView {
	t.Helper()

	s := sim.NewState(1, 0, 0)
	s.Debt = b.Credit.MaxPrincipal
	s.LifetimeEarned = sim.Coins(b.Progression.PrestigeDivisor - 1)

	if _, ok := s.AddTank(sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres); !ok {
		t.Fatal("sem tanque")
	}

	return viewOf(Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}, b, newPlans())
}

func TestAViewDoTanqueCarregaOQueOPainelLe(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	tank, view := viewOfStocked(t, &b)

	if tank.MaxBatches != sim.MaxBatchesPerTank {
		t.Errorf("max_batches saiu %d, o dominio diz %d", tank.MaxBatches, sim.MaxBatchesPerTank)
	}
	if tank.Batches != 1 {
		t.Errorf("batch_count saiu %d num tanque com um lote", tank.Batches)
	}
	if view.Broke {
		t.Error("uma fazenda com peixe e caixa nao esta quebrada")
	}
	if !brokeView(t, &b).Broke {
		t.Error("uma fazenda sem peixe, sem caixa e sem credito tem que sair como quebrada")
	}
	if tank.Decision.CostPerDay <= 0 {
		t.Error("cost_per_day_cents saiu zerado: manutencao e energia sempre custam algo")
	}
	if tank.Decision.HoldToGrams > 0 && tank.Decision.HoldCostCents <= 0 {
		t.Error("hold_cost_cents saiu zerado numa projecao que dura dias")
	}
}

func TestARacaoPorDiaVemDaMassaComidaENaoDeDinheiro(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	tank, _ := viewOfStocked(t, &b)
	d := tank.Decision

	if d.FeedPerDayG <= 0 {
		t.Fatal("feed_per_day_grams saiu zerado")
	}

	biomass := int64(tank.Fish) * tank.MeanGrams
	percent := d.FeedPerDayG * 100 / biomass

	if percent < 1 || percent > 6 {
		t.Errorf("racao de %d g/dia para %d g de biomassa e %d%% do peso corporal, fora da faixa real",
			d.FeedPerDayG, biomass, percent)
	}

	byMoney := d.CostPerDay * gramsPerKilo / int64(sim.MarketAt(&b, 0).FeedKg)
	if d.FeedPerDayG == byMoney {
		t.Errorf("feed_per_day_grams bate com dinheiro dividido por preco (%d): voltou a inverter o custo", byMoney)
	}
}

func TestTodoTipoDeTanqueSaiNaViewComNomeELotacao(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range sim.TankKindNames() {
		kind, ok := sim.TankKindNamed(name)
		if !ok {
			t.Fatalf("%q nao volta para nenhum tipo de tanque", name)
		}

		s := sim.NewState(1, 0, 0)
		if _, ok := s.AddTank(kind, b.Tanks[kind].Litres); !ok {
			t.Fatalf("%s: sem tanque", name)
		}

		tank := viewOf(Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}, &b, newPlans()).Tanks[0]
		if tank.Kind != name {
			t.Errorf("o tanque %s saiu como %q na API", name, tank.Kind)
		}
		if tank.Capacity <= 0 {
			t.Errorf("o tanque %s saiu sem lotacao: falta a linha dele no balance.toml", name)
		}
	}
}

func TestActionOfMapeiaCadaNomeParaOSeuKind(t *testing.T) {
	t.Parallel()

	esperado := map[string]sim.ActionKind{
		"feed": sim.ActionFeed, "buy_feed": sim.ActionBuyFeed, "aerate": sim.ActionAerate,
		"harvest": sim.ActionHarvest, "stock": sim.ActionStock, "buy_tank": sim.ActionBuyTank,
		"buy_upgrade": sim.ActionBuyUpgrade, "treat": sim.ActionTreat,
		"prestige": sim.ActionPrestige, "restart": sim.ActionRestart,
		"borrow": sim.ActionBorrow, "repay": sim.ActionRepay,
	}

	if len(esperado) != len(actionKindByName) {
		t.Fatalf("o enum tem %d nomes e o teste conhece %d: alguem acrescentou acao sem cobrir",
			len(actionKindByName), len(esperado))
	}

	for name, want := range esperado {
		got, err := actionOf(actionBody{
			Kind: name, Tank: 1, Amount: 1, TankKind: "viveiro_escavado", Auto: "comedouro",
		})
		if err != nil {
			t.Errorf("%s: actionOf recusou: %v", name, err)

			continue
		}
		if got.Kind != want {
			t.Errorf("%s virou %v, queria %v", name, got.Kind, want)
		}
	}
}

func TestActionsQueNaoSaoDeTanqueNaoExigemTanque(t *testing.T) {
	t.Parallel()

	for _, kind := range []sim.ActionKind{sim.ActionPrestige, sim.ActionRestart, sim.ActionBorrow, sim.ActionRepay} {
		if needsTank(kind) {
			t.Errorf("%v nao age sobre um tanque, mas needsTank pede um", kind)
		}
	}
}
