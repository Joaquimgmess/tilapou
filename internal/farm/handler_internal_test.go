package farm

import (
	"errors"
	"testing"
	"time"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func viewOfStocked(t *testing.T, b *sim.Balance) (api.Tank, api.Snapshot) {
	t.Helper()

	s := sim.NewState(1, 0, 0)
	s.Cash = 5_000_000

	id, ok := s.AddTank(b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
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

// frontBatch is the batch the old api.Tank used to flatten into itself.
func frontBatch(t *testing.T, tank api.Tank) api.Batch {
	t.Helper()

	if len(tank.Batches) == 0 {
		t.Fatal("o tanque saiu sem lote")
	}

	return tank.Batches[0]
}

func brokeView(t *testing.T, b *sim.Balance) api.Snapshot {
	t.Helper()

	s := sim.NewState(1, 0, 0)
	s.Debt = b.Credit.MaxPrincipal
	s.LifetimeEarned = sim.Coins(b.Progression.PrestigeDivisor - 1)

	if _, ok := s.AddTank(b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres); !ok {
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
	if tank.BatchCount != 1 {
		t.Errorf("batch_count saiu %d num tanque com um lote", tank.BatchCount)
	}
	if view.Broke {
		t.Error("uma fazenda com peixe e caixa nao esta quebrada")
	}
	if !brokeView(t, &b).Broke {
		t.Error("uma fazenda sem peixe, sem caixa e sem credito tem que sair como quebrada")
	}
	if frontBatch(t, tank).Decision.CostPerDay <= 0 {
		t.Error("cost_per_day_cents saiu zerado: manutencao e energia sempre custam algo")
	}
	if frontBatch(t, tank).Decision.HoldToGrams > 0 && frontBatch(t, tank).Decision.HoldCostCents <= 0 {
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
	d := frontBatch(t, tank).Decision

	if d.FeedPerDayG <= 0 {
		t.Fatal("feed_per_day_grams saiu zerado")
	}

	biomass := int64(tank.Fish) * frontBatch(t, tank).MeanGrams
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
		if _, ok := s.AddTank(&b, kind, b.Tanks[kind].Litres); !ok {
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

	if nomes := sim.ActionKindNames(); len(esperado) != len(nomes) {
		t.Fatalf("o enum tem %d nomes e o teste conhece %d: alguem acrescentou acao sem cobrir",
			len(nomes), len(esperado))
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

func TestNeedsTankSeparaAsAcoesDeTanqueDasDemais(t *testing.T) {
	t.Parallel()

	comTanque := []sim.ActionKind{
		sim.ActionFeed, sim.ActionBuyFeed, sim.ActionAerate, sim.ActionHarvest,
		sim.ActionStock, sim.ActionBuyUpgrade, sim.ActionTreat,
	}
	semTanque := []sim.ActionKind{
		sim.ActionBuyTank, sim.ActionPrestige, sim.ActionRestart,
		sim.ActionBorrow, sim.ActionRepay, sim.ActionUnknown,
	}

	if total := len(comTanque) + len(semTanque); total != len(sim.ActionKindNames())+1 {
		t.Fatalf("o enum tem %d acoes mais ActionUnknown e o teste conhece %d: alguem acrescentou acao sem cobrir",
			len(sim.ActionKindNames()), total)
	}

	for _, kind := range comTanque {
		if !needsTank(kind) {
			t.Errorf("%v age sobre um tanque, mas needsTank aceita sem tank_id", kind)
		}
	}
	for _, kind := range semTanque {
		if needsTank(kind) {
			t.Errorf("%v nao age sobre um tanque, mas needsTank pede um", kind)
		}
	}
}

func TestActionOfRecusaCampoObrigatorioFaltando(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		body actionBody
		want error
	}{
		"kind desconhecido":         {actionBody{Kind: "voar"}, ErrUnknownAction},
		"kind vazio":                {actionBody{}, ErrUnknownAction},
		"unknown nao e acao":        {actionBody{Kind: "unknown"}, ErrUnknownAction},
		"feed sem tanque":           {actionBody{Kind: "feed"}, ErrMissingTank},
		"harvest sem tanque":        {actionBody{Kind: "harvest"}, ErrMissingTank},
		"stock sem tanque":          {actionBody{Kind: "stock"}, ErrMissingTank},
		"buy_upgrade sem auto":      {actionBody{Kind: "buy_upgrade", Tank: 1}, ErrMissingAuto},
		"buy_upgrade auto invalido": {actionBody{Kind: "buy_upgrade", Tank: 1, Auto: "trator"}, ErrMissingAuto},
		"buy_tank sem kind":         {actionBody{Kind: "buy_tank"}, ErrMissingTankKind},
		"buy_tank kind invalido":    {actionBody{Kind: "buy_tank", TankKind: "aquario"}, ErrMissingTankKind},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := actionOf(tc.body); !errors.Is(err, tc.want) {
				t.Errorf("actionOf devolveu %v, queria %v", err, tc.want)
			}
		})
	}
}

func TestTodoLoteDoTanqueApareceNaView(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	s := sim.NewState(1, 0, 0)
	s.Cash = 10_000_000

	id, ok := s.AddTank(&b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 276, 450*sim.MicrogramsPerGram, 200_000)
	s.StockTank(id, 147, 120*sim.MicrogramsPerGram, 90_000)
	s.SeedOxygen(&b)

	view := viewOf(Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}, &b, newPlans())
	if len(view.Tanks) != 1 {
		t.Fatalf("a view saiu com %d tanques", len(view.Tanks))
	}

	tank := view.Tanks[0]
	if len(tank.Batches) != 2 {
		t.Fatalf("o tanque tem 2 lotes e a view mostra %d", len(tank.Batches))
	}

	var fish int32
	for i := range tank.Batches {
		fish += tank.Batches[i].Fish
	}

	if fish != tank.Fish {
		t.Errorf("os lotes somam %d peixes e o tanque diz %d: a view esconde lote", fish, tank.Fish)
	}
}

// Todo snapshot monta a view, e a view pede conselho de lotacao por tanque. Se esse conselho
// simular um ciclo inteiro por tanque, cada requisicao HTTP paga segundos de simulacao —
// tempo que o cache de planos da fatia existe justamente para nao pagar duas vezes.
func TestMontarAViewNaoSimulaUmCicloPorTanque(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	s := sim.NewState(1, 0, 0)
	s.Cash = 50_000_000

	for _, kind := range []sim.TankKind{
		sim.TankEarthPond, sim.TankNetCage, sim.TankBiofloc, sim.TankRecirculation,
	} {
		id, ok := s.AddTank(&b, kind, b.Tanks[kind].Litres)
		if !ok {
			t.Fatalf("sem tanque do tipo %s", kind)
		}
		s.StockTank(id, 100, b.Growth.FingerlingMass, 8_000)
	}
	s.SeedOxygen(&b)

	snap := Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}
	p := newPlans()

	// O caminho quente e o snapshot repetido: o primeiro ainda pode simular para montar o
	// plano do dia, o segundo tem de sair do cache.
	viewOf(snap, &b, p)

	start := time.Now()
	viewOf(snap, &b, p)

	const budget = 50 * time.Millisecond
	if took := time.Since(start); took > budget {
		t.Errorf("montar a view de %d tanques levou %s, acima de %s: alguem esta simulando um ciclo fora do cache",
			s.TankCount, took, budget)
	}
}

// O zero value de sim.Plans vale "nenhum ciclo fecha", e com ele o peao de despesca vende no
// peso minimo em vez do peso de melhor margem. Nada no compilador cobra que a sessao encha os
// planos antes de adiantar a simulacao, entao e cobrado aqui.
func TestOCacheMontaOPlanoDeCadaTanqueQueAFazendaTem(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	s := sim.NewState(1, 0, 0)
	s.Cash = 50_000_000

	owned := []sim.TankKind{sim.TankEarthPond, sim.TankBiofloc}
	for _, kind := range owned {
		if _, ok := s.AddTank(&b, kind, b.Tanks[kind].Litres); !ok {
			t.Fatalf("sem tanque do tipo %s", kind)
		}
	}

	p := newPlans()

	got := p.forFarm(&b, &s)
	for _, kind := range owned {
		if got[kind].Days <= 0 {
			t.Errorf("o tanque %s foi sem plano: o peao vai vender no peso minimo", kind)
		}
	}
	if got[sim.TankNetCage].Days != 0 {
		t.Error("foi plano de um tipo de tanque que a fazenda nao tem: cada um custa duas simulacoes de ciclo")
	}
}
