// Package farm holds a player's farm: persisted state, the session that advances it
// up to now and the HTTP routes.
package farm

import (
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const (
	startingCash = sim.Coins(50_000)
	startingFish = sim.FishCount(2_000)
	startingMass = 450 * sim.MicrogramsPerGram
	startingFeed = 200 * sim.MicrogramsPerKilogram
)

// ID identifies a farm.
type ID = uuid.UUID

// Farm holds the sim state with the origin of the ticks (Epoch) and the Revision that
// detects concurrent writes.
type Farm struct {
	ID        ID
	PlayerID  uuid.UUID
	Name      string
	Epoch     time.Time
	Revision  int64
	State     sim.State
	CreatedAt time.Time
}

// TickAt counts one tick per second since epoch, and 0 if now is earlier than it.
func TickAt(epoch, now time.Time) sim.Tick {
	if now.Before(epoch) {
		return 0
	}

	return sim.Tick(now.Sub(epoch) / time.Second)
}

// New builds the farm at tick 0, with starting cash and an already stocked earthen pond.
func New(id, playerID uuid.UUID, name string, epoch time.Time, zone sim.ZoneOffset, seed sim.Seed, b *sim.Balance) Farm {
	state := sim.NewState(seed, zone, 0)
	state.Cash = startingCash
	state.BalanceVersion = b.Version

	tank, ok := state.AddTank(b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
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
