package farm

import (
	"sync"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

type batchKey struct {
	tank  sim.TankID
	batch sim.BatchID
	fish  sim.FishCount
	grams int64
	day   int64
}

type plans struct {
	mu        sync.Mutex
	day       int64
	zone      sim.ZoneOffset
	cache     map[sim.TankKind]sim.CyclePlan
	decisions map[batchKey]DecisionView
}

func newPlans() *plans {
	return &plans{
		cache:     make(map[sim.TankKind]sim.CyclePlan),
		decisions: make(map[batchKey]DecisionView),
	}
}

func (p *plans) decision(key batchKey, compute func() DecisionView) DecisionView {
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
