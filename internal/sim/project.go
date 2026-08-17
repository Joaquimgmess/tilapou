package sim

// Projection e o retrato dos numeros de topo do save num tick.
type Projection struct {
	Tick     Tick
	Cash     Coins
	Lifetime Coins
	Biomass  Micrograms
	Fish     FishCount
	Tanks    int32
	Prestige uint32
}

// Project extrai o retrato atual do estado sem avancar a simulacao.
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
