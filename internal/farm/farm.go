package farm

import (
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/catalog/internal/sim"
)

type ID = uuid.UUID

type Farm struct {
	ID        ID
	PlayerID  uuid.UUID
	Name      string
	Epoch     time.Time
	Revision  int64
	State     sim.State
	CreatedAt time.Time
}

func TickAt(epoch, now time.Time) sim.Tick {
	if now.Before(epoch) {
		return 0
	}

	return sim.Tick(now.Sub(epoch) / time.Second)
}

func New(id, playerID uuid.UUID, name string, epoch time.Time, zone sim.ZoneOffset, seed sim.Seed, b *sim.Balance) Farm {
	state := sim.NewState(seed, zone, 0)
	state.Cash = startingCash
	state.BalanceVersion = b.Version

	tank, ok := state.AddTank(sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
	if ok {
		state.StockTank(tank, startingFish, startingMass)
		state.LoadFeed(tank, startingFeed)
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
	startingMass = 300 * sim.MicrogramsPerGram
	startingFeed = 200 * sim.MicrogramsPerKilogram
)
