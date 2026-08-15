package farm_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/farm"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func TestStartingBatchCarriesWhatItCostToRaise(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	f := farm.New(uuid.New(), uuid.New(), "t", time.Unix(0, 0), 0, 1, &b)
	tank := f.State.Tanks[0]
	batch := tank.Batches[0]

	if batch.Cost <= 0 {
		t.Fatal("lote inicial sem custo acumulado")
	}

	value := sim.Coins(int64(b.PriceFor(batch.MeanMass, 0)) *
		int64(batch.Biomass()) / int64(sim.MicrogramsPerKilogram))
	if batch.Cost >= value {
		t.Fatalf("lote inicial nasce no prejuizo: custo %d, valor %d", batch.Cost, value)
	}
}
