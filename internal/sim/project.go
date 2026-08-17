package sim

// Projection is the snapshot of the save's top-level numbers at a tick.
type Projection struct {
	Tick     Tick
	Cash     Coins
	Lifetime Coins
	Biomass  Micrograms
	Fish     FishCount
	Tanks    int32
	Prestige uint32
}

// Project extracts the current snapshot of the state without advancing the simulation.
func Project(s *State) Projection {
	return Projection{
		Tick:     s.Tick,
		Cash:     s.Cash,
		Lifetime: s.LifetimeEarned,
		Biomass:  s.Biomass(),
		Fish:     s.Fish(),
		Tanks:    s.TankCount,
		Prestige: s.Prestige,
	}
}
