package farm

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func stockedForDecision(t *testing.T, b *sim.Balance) sim.State {
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

	return s
}

// checkCoherent holds for every view: the derived fields must match the raw fields of the
// same payload, whatever the cache did.
func checkCoherent(t *testing.T, step string, tv api.Tank) {
	t.Helper()

	if len(tv.Batches) == 0 {
		t.Fatalf("%s: o tanque saiu sem lote", step)
	}

	bv := tv.Batches[0]
	d := bv.Decision
	if d.FeedPerDayG > 0 {
		want := tv.FeedKg * gramsPerKilo / d.FeedPerDayG
		if d.DaysOfFeed != want {
			t.Errorf("%s: days_of_feed = %d, mas %d kg a %d g/d dao %d",
				step, d.DaysOfFeed, tv.FeedKg, d.FeedPerDayG, want)
		}
	}

	if want := d.HoldCents - bv.CostCents - d.HoldCostCents; d.HoldMargin != want {
		t.Errorf("%s: hold_margin = %d, mas %d de venda menos %d de custo menos %d de gasto dao %d",
			step, d.HoldMargin, d.HoldCents, bv.CostCents, d.HoldCostCents, want)
	}

	kilos := int64(bv.Fish) * bv.MeanGrams / gramsPerKilo
	if want := bv.PriceKgCents * kilos; d.SellNowCents != want {
		t.Errorf("%s: sell_now_cents = %d, mas %d kg a %d c/kg dao %d",
			step, d.SellNowCents, kilos, bv.PriceKgCents, want)
	}
}

func TestDecisaoBateComOsNumerosCrusDoMesmoPayload(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	s := stockedForDecision(t, &b)
	p := newPlans()
	tank := &s.Tanks[0]

	steps := []struct {
		name string
		do   func()
	}{
		{"inicio", func() {}},
		{"compra 800 kg de racao", func() { s.LoadFeed(tank.ID, 800*sim.MicrogramsPerKilogram, sim.MarketAt(&b, s.Tick).FeedKg) }},
		{"avanca 10 ticks no mesmo dia", func() { s.Tick += 10 }},
		{"compra mais 500 kg", func() { s.LoadFeed(tank.ID, 500*sim.MicrogramsPerKilogram, sim.MarketAt(&b, s.Tick).FeedKg) }},
		{"vira o dia", func() { s.Tick += sim.TicksPerDay }},
	}

	seen := make(map[int64]bool)

	for _, step := range steps {
		step.do()

		view := viewOf(Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}, &b, p)
		if len(view.Tanks) == 0 {
			t.Fatalf("%s: a view saiu sem tanque", step.name)
		}

		checkCoherent(t, step.name, view.Tanks[0])
		seen[view.Tanks[0].Batches[0].Decision.DaysOfFeed] = true
	}

	if len(seen) < 2 {
		t.Errorf("comprar racao nunca mudou days_of_feed: %v", seen)
	}
}

func TestChaveDaDecisaoNaoMudaATodoTick(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	s := stockedForDecision(t, &b)
	prev := newDecisionInput(&s, &s.Tanks[0], &s.Tanks[0].Batches[0])

	// O cache paga 33 ms por miss, entao a chave pode mudar algumas vezes por dia, nao a
	// cada tick: um campo que churna por tick zera o cache e volta a custar 26% de um core.
	const budget = 48

	changes := 0
	for range int(sim.TicksPerDay) {
		out, advanceErr := sim.Advance(sim.Input{State: s, Until: s.Tick + 1, Balance: &b})
		if advanceErr != nil {
			t.Fatalf("avancando um tick: %v", advanceErr)
		}
		s = out.State

		in := newDecisionInput(&s, &s.Tanks[0], &s.Tanks[0].Batches[0])
		if in != prev {
			changes++
		}
		prev = in
	}

	if changes > budget {
		t.Errorf("a chave da decisao mudou %d vezes em um dia, o teto e %d: o cache morreu", changes, budget)
	}
}
