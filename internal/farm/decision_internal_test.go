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

	// A coluna VALOR sai da mesma conta: era ela que truncava a biomassa em quilos inteiros,
	// congelando o numero num degrau de 1 kg — e, depois que a despesca passou a decidir por
	// valor, chegando a mostrar o lado errado do piso que diz se a fazenda tem jogada.
	if want := int64(sim.GrossValue(sim.Coins(bv.PriceKgCents), massOf(bv))); bv.ValueCents != want {
		t.Errorf("%s: value_cents = %d, mas %d peixes de %d g a %d c/kg dao %d",
			step, bv.ValueCents, bv.Fish, bv.MeanGrams, bv.PriceKgCents, want)
	}

	// A conferencia sai da conta do dominio, e nao de uma copia da linha do handler: recopiar
	// a formula fazia este teste ser tautologia — ele nao podia falhar, e foi por isso que a
	// divergencia da coluna VALOR viveu ate o @qa medi-la contra o caixa.
	if want := int64(sim.GrossValue(sim.Coins(bv.PriceKgCents), massOf(bv))); d.SellNowCents != want {
		t.Errorf("%s: sell_now_cents = %d, mas %d peixes de %d g a %d c/kg dao %d",
			step, d.SellNowCents, bv.Fish, bv.MeanGrams, bv.PriceKgCents, want)
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

// massOf e a biomassa do lote em micrograma, a partir do que o payload publica.
func massOf(bv api.Batch) sim.Micrograms {
	return sim.Micrograms(int64(bv.Fish) * bv.MeanGrams * int64(sim.MicrogramsPerGram))
}

// O lote do outro teste tem 405 kg cravados, e ai truncar em quilos inteiros da o mesmo
// numero — foi por isso que a mutacao sobreviveu quando eu tentei. Aqui a biomassa NAO fecha
// em quilo, que e a faixa em que a coluna congelava: de 370 a 380 peixes ela imprimia o mesmo
// valor, e chegava a mostrar o lado errado do piso que decide se a fazenda tem jogada.
func TestAColunaValorNaoTruncaEmQuilos(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	vistos := make(map[int64]bool)

	for fish := int32(370); fish <= 380; fish++ {
		s := stockedForDecision(t, &b)
		batch := &s.Tanks[0].Batches[0]
		batch.Fish, batch.MeanMass = sim.FishCount(fish), 24*sim.MicrogramsPerGram

		view := viewOf(Snapshot{Farm: Farm{State: s}, Projection: sim.Project(&s)}, &b, newPlans())
		bv := view.Tanks[0].Batches[0]

		want := int64(sim.GrossValue(sim.Coins(bv.PriceKgCents), batch.Biomass()))
		if bv.ValueCents != want {
			t.Errorf("%d peixes: a coluna VALOR diz %d e a conta que a despesca credita da %d",
				fish, bv.ValueCents, want)
		}

		vistos[bv.ValueCents] = true
	}

	if len(vistos) < 11 {
		t.Errorf("onze lotes diferentes imprimiram %d valores distintos: a coluna esta congelada num degrau de quilo", len(vistos))
	}
}
