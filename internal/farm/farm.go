// Package farm guarda a fazenda de um jogador: estado persistido, sessao que o
// adianta ate agora e rotas HTTP.
package farm

import (
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

// ID identifica uma fazenda.
type ID = uuid.UUID

// Farm guarda o estado do sim com a origem dos ticks (Epoch) e a Revision que
// detecta escrita concorrente.
type Farm struct {
	ID        ID
	PlayerID  uuid.UUID
	Name      string
	Epoch     time.Time
	Revision  int64
	State     sim.State
	CreatedAt time.Time
}

// TickAt conta um tick por segundo desde epoch, e 0 se now for anterior a ele.
func TickAt(epoch, now time.Time) sim.Tick {
	if now.Before(epoch) {
		return 0
	}

	return sim.Tick(now.Sub(epoch) / time.Second)
}

// New monta a fazenda no tick 0, com caixa inicial e um viveiro ja povoado.
func New(id, playerID uuid.UUID, name string, epoch time.Time, zone sim.ZoneOffset, seed sim.Seed, b *sim.Balance) Farm {
	state := sim.NewState(seed, zone, 0)
	state.Cash = startingCash
	state.BalanceVersion = b.Version

	tank, ok := state.AddTank(sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
	if ok {
		state.StockTank(tank, startingFish, startingMass, sim.RaisingCost(b, startingFish, startingMass, 0))
		state.LoadFeed(tank, startingFeed, sim.MarketAt(b, 0).FeedKg)
		state.SeedOxygen(b)
	}

	return Farm{
		ID:        id,
		PlayerID:  playerID,
		Name:      name,
		Epoch:     epoch,
		Revision:  0,
		State:     state,
		CreatedAt: epoch,
	}
}

const (
	startingCash = sim.Coins(50_000)
	startingFish = sim.FishCount(2_000)
	startingMass = 450 * sim.MicrogramsPerGram
	startingFeed = 200 * sim.MicrogramsPerKilogram
)
