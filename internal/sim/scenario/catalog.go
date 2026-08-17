package scenario

import "github.com/Joaquimgmess/tilapou/internal/sim"

const ticksPerHour = sim.Tick(60)

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

type pond struct {
	kind   sim.TankKind
	litres sim.Litres
	fish   sim.FishCount
	mass   sim.Micrograms
	feedKg int64
}

func stock(state *sim.State, p pond) {
	id, ok := state.AddTank(p.kind, p.litres)
	if !ok {
		return
	}

	state.StockTank(id, p.fish, p.mass, 0)
	state.LoadFeed(id, sim.Micrograms(p.feedKg)*sim.MicrogramsPerKilogram, 0)
}

type schedule struct {
	everyHours int64
	forDays    int64
}

func feedEvery(s schedule) []sim.Action {
	last := sim.Tick(s.forDays) * sim.TicksPerDay
	step := sim.Tick(s.everyHours) * ticksPerHour

	actions := make([]sim.Action, 0, last/step+1)

	var id sim.ActionID
	for tick := sim.Tick(1); tick <= last; tick += step {
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 2_000, mass: 300 * sim.MicrogramsPerGram, feedKg: 3_000})
		},
		Actions: feedEvery(schedule{everyHours: 6, forDays: 90}),
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 2_000, mass: 300 * sim.MicrogramsPerGram, feedKg: 40})
		},
		Actions: feedEvery(schedule{everyHours: 6, forDays: 60}),
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 12_000, mass: 600 * sim.MicrogramsPerGram, feedKg: 5_000})
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 12_000, mass: 600 * sim.MicrogramsPerGram, feedKg: 5_000})
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
			stock(s, pond{kind: sim.TankNetCage, litres: 6_000, fish: 900, mass: 300 * sim.MicrogramsPerGram, feedKg: 2_000})
		},
		Actions: feedEvery(schedule{everyHours: 6, forDays: 30}),
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 2_000, mass: 300 * sim.MicrogramsPerGram, feedKg: 3_000})
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 2_000, mass: 300 * sim.MicrogramsPerGram, feedKg: 20})
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
			stock(s, pond{kind: sim.TankEarthPond, litres: 1_000_000, fish: 500, mass: 200 * sim.MicrogramsPerGram, feedKg: 100})
		},
		Actions: []sim.Action{
			{ID: 7, Kind: sim.ActionBuyTank, At: 2 * sim.TicksPerDay, TankKind: sim.TankRecirculation},
		},
	}
}
