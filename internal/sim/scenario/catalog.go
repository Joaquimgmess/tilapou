package scenario

import "github.com/Joaquimgmess/catalog/internal/sim"

func All() []Scenario {
	return []Scenario{
		heranca(),
		comedouroSeca(),
		densidadeContraOxigenio(),
		aeradorSalva(),
		tanqueRede(),
		comedouroAutomatico(),
		acaoRejeitadaNoCatchUp(),
	}
}

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

	state.StockTank(id, fish, mass)
	state.LoadFeed(id, sim.Micrograms(feedKg)*sim.MicrogramsPerKilogram)
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
			{ID: 1, Kind: sim.ActionBuyUpgrade, At: 1, Auto: sim.AutoFeeder},
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
