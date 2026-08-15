package farm

import (
	"sync"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

type plans struct {
	mu    sync.Mutex
	day   int64
	zone  sim.ZoneOffset
	cache map[sim.TankKind]sim.CyclePlan
}

func newPlans() *plans {
	return &plans{cache: map[sim.TankKind]sim.CyclePlan{}}
}

func (p *plans) at(b *sim.Balance, kind sim.TankKind, tick sim.Tick, zone sim.ZoneOffset) sim.CyclePlan {
	day := int64(tick / sim.TicksPerDay)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.day != day || p.zone != zone {
		p.day, p.zone = day, zone
		clear(p.cache)
	}
	if plan, ok := p.cache[kind]; ok {
		return plan
	}

	plan := b.CycleAt(kind, sim.Tick(day)*sim.TicksPerDay, zone)
	p.cache[kind] = plan

	return plan
}
