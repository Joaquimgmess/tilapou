package farm

import (
	"sync"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

// decisionInput is the whole input of a decision, and therefore its cache key: what is
// not here cannot change the result. The tank comes quantized by newDecisionInput.
type decisionInput struct {
	tank     sim.Tank
	batch    sim.BatchID
	day      int64
	zone     sim.ZoneOffset
	prestige uint32
}

// newDecisionInput quantizes what churns every tick so the cache survives the day: the
// accrued cost is left out because the projection only adds to it, and the caller
// subtracts the current one. The
// tank literal has no field names on purpose: a new field in sim.Tank stops compiling
// here until someone decides whether it goes raw, quantized or zeroed.
func newDecisionInput(state *sim.State, tank *sim.Tank, batch *sim.Batch) decisionInput {
	//nolint:govet // composites: literal sem nomes e a trava; campo novo em sim.Tank para aqui
	quantized := sim.Tank{
		tank.ID,
		tank.Kind,
		tank.Litres,
		quantizedBatches(batch),
		1,
		tank.FeedStock / sim.MicrogramsPerKilogram * sim.MicrogramsPerKilogram,
		0,
		tank.Upgrades,
		tank.Oxygen,
		tank.Aerating,
		0,
		tank.FeedUnitCost,
		0,
		tank.CarrierUntil,
		sim.Accrual{},
	}

	return decisionInput{
		tank:     quantized,
		batch:    batch.ID,
		day:      int64(state.Tick / sim.TicksPerDay),
		zone:     state.Zone,
		prestige: state.Prestige,
	}
}

func (in decisionInput) forecast(b *sim.Balance, target sim.Micrograms) sim.Forecast {
	return sim.ForecastAhead(b, sim.ForecastInput{
		Tank:     in.tank,
		Batch:    in.batch,
		At:       sim.Tick(in.day) * sim.TicksPerDay,
		Zone:     in.zone,
		Prestige: in.prestige,
		Target:   target,
	})
}

func quantizedBatches(batch *sim.Batch) [sim.MaxBatchesPerTank]sim.Batch {
	var only [sim.MaxBatchesPerTank]sim.Batch

	//nolint:govet // composites: literal sem nomes e a trava; campo novo em sim.Batch para aqui
	only[0] = sim.Batch{
		batch.ID,
		batch.Fish,
		batch.MeanMass / sim.MicrogramsPerGram * sim.MicrogramsPerGram,
		0,
		0,
		batch.StockedAt,
		0,
		0,
		0,
		0,
		batch.Sick,
		0,
		0,
	}
	return only
}

type plans struct {
	mu        sync.Mutex
	day       int64
	zone      sim.ZoneOffset
	cache     map[sim.TankKind]sim.CyclePlan
	decisions map[decisionInput]DecisionView
}

func newPlans() *plans {
	return &plans{
		cache:     make(map[sim.TankKind]sim.CyclePlan),
		decisions: make(map[decisionInput]DecisionView),
	}
}

func (p *plans) decision(key decisionInput, compute func() DecisionView) DecisionView {
	p.mu.Lock()
	if view, ok := p.decisions[key]; ok {
		p.mu.Unlock()

		return view
	}
	p.mu.Unlock()

	view := compute()

	p.mu.Lock()
	defer p.mu.Unlock()

	if key.day != p.day {
		clear(p.decisions)
		p.day = key.day
	}
	p.decisions[key] = view

	return view
}

func (p *plans) at(b *sim.Balance, kind sim.TankKind, tick sim.Tick, zone sim.ZoneOffset) sim.CyclePlan {
	day := int64(tick / sim.TicksPerDay)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.day != day || p.zone != zone {
		p.day, p.zone = day, zone
		clear(p.cache)
		clear(p.decisions)
	}
	if plan, ok := p.cache[kind]; ok {
		return plan
	}

	plan := b.CycleAt(kind, sim.Tick(day)*sim.TicksPerDay, zone)
	p.cache[kind] = plan

	return plan
}
