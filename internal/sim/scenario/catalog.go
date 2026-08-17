package scenario

import "github.com/Joaquimgmess/tilapou/internal/sim"

// All builds the catalog scenarios from scratch on every call.
func All() []Scenario {
	return []Scenario{
		heranca(),
		comedouroSeca(),
		densidadeContraOxigenio(),
		aeradorSalva(),
		tanqueRede(),
		semTrato(),
		comedouroAutomatico(),
		acaoRejeitadaNoCatchUp(),
	}
}

// ByName returns a false bool, with a zeroed Scenario, when the name does not exist.
func ByName(name string) (Scenario, bool) {
	for _, s := range All() {
		if s.Name == name {
			return s, true
		}
	}

	return Scenario{}, false
}

func stock(state *sim.State, kind sim.TankKind, litres sim.Litres, fish sim.FishCount, mass sim.Micrograms, feedKg int64) {
	id, ok := state.AddTank(kind, litres)
	if !ok {
		return
	}

	state.StockTank(id, fish, mass, 0)
	state.LoadFeed(id, sim.Micrograms(feedKg)*sim.MicrogramsPerKilogram, 0)
}

func feedEvery(hours, days int64) []sim.Action {
	var (
		actions []sim.Action
		id      sim.ActionID
	)

	for tick := sim.Tick(1); tick <= sim.Tick(days)*sim.TicksPerDay; tick += sim.Tick(hours) * 60 {
		id++
		actions = append(actions, sim.Action{ID: id, Kind: sim.ActionFeed, At: tick, Tank: 1})
	}

	return actions
}

func heranca() Scenario {
	return Scenario{
		Name: "heranca",
		Zone: -180,
		Seed: 1,
		Days: 90,
		Cash: 500_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 2_000, 300*sim.MicrogramsPerGram, 3_000)
		},
		Actions: feedEvery(6, 90),
	}
}

func comedouroSeca() Scenario {
	return Scenario{
		Name: "comedouro-seca",
		Zone: -180,
		Seed: 2,
		Days: 60,
		Cash: 500_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 2_000, 300*sim.MicrogramsPerGram, 40)
		},
		Actions: feedEvery(6, 60),
	}
}

func densidadeContraOxigenio() Scenario {
	return Scenario{
		Name: "densidade-contra-oxigenio",
		Zone: -180,
		Seed: 3,
		Days: 5,
		Cash: 500_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 12_000, 600*sim.MicrogramsPerGram, 5_000)
		},
	}
}

func aeradorSalva() Scenario {
	return Scenario{
		Name: "aerador-salva",
		Zone: -180,
		Seed: 3,
		Days: 5,
		Cash: 500_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 12_000, 600*sim.MicrogramsPerGram, 5_000)
		},
		Actions: []sim.Action{
			{ID: 1, Kind: sim.ActionAerate, At: 1, Tank: 1, Amount: 1},
		},
	}
}

func tanqueRede() Scenario {
	return Scenario{
		Name: "tanque-rede",
		Zone: -180,
		Seed: 5,
		Days: 30,
		Cash: 500_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankNetCage, 6_000, 900, 300*sim.MicrogramsPerGram, 2_000)
		},
		Actions: feedEvery(6, 30),
	}
}

func semTrato() Scenario {
	return Scenario{
		Name: "sem-trato",
		Zone: -180,
		Seed: 7,
		Days: 20,
		Cash: 500_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 2_000, 300*sim.MicrogramsPerGram, 3_000)
		},
	}
}

func comedouroAutomatico() Scenario {
	return Scenario{
		Name: "comedouro-automatico",
		Zone: -180,
		Seed: 6,
		Days: 45,
		Cash: 5_000_000,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 2_000, 300*sim.MicrogramsPerGram, 20)
		},
		Actions: []sim.Action{
			{ID: 1, Kind: sim.ActionBuyUpgrade, At: 1, Tank: 1, Auto: sim.AutoFeeder},
		},
	}
}

func acaoRejeitadaNoCatchUp() Scenario {
	return Scenario{
		Name: "acao-rejeitada-no-catchup",
		Zone: -180,
		Seed: 4,
		Days: 3,
		Cash: 100,
		Setup: func(s *sim.State) {
			stock(s, sim.TankEarthPond, 1_000_000, 500, 200*sim.MicrogramsPerGram, 100)
		},
		Actions: []sim.Action{
			{ID: 7, Kind: sim.ActionBuyTank, At: 2 * sim.TicksPerDay, TankKind: sim.TankRecirculation},
		},
	}
}
