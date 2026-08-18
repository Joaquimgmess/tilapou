package balance_test

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func TestLoadQuantizesTheShippedFile(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := map[string]struct {
		got, want int64
	}{
		"versao":              {int64(b.Version), 1},
		"tgc em ppm":          {int64(b.Growth.TGCPPM), 1_400},
		"temperatura de ref":  {int64(b.Growth.ReferenceTemp), 26_000},
		"peso maximo em ug":   {int64(b.Growth.MaxMass), 1_100_000_000},
		"peso de abate em ug": {int64(b.Growth.HarvestMass), 600_000_000},
		"od critico":          {int64(b.Water.Critical), 2_000},
		"fator de custo":      {int64(b.Progression.CostFactorPPM), 1_120_000},
		"passos de racao":     {int64(b.Ration.Len), 5},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", name, tt.got, tt.want)
			}
		})
	}
}

func TestLoadFillsEveryTankSlot(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, kind := range []sim.TankKind{sim.TankEarthPond, sim.TankNetCage, sim.TankBiofloc, sim.TankRecirculation} {
		spec := b.Tanks[kind]
		if spec.Litres <= 0 || spec.MaxDensityPerM3 <= 0 || spec.BaseCost <= 0 {
			t.Errorf("tanque %d ficou sem spec: %+v", kind, spec)
		}
	}
}

func TestLoadIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	second, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if first != second {
		t.Error("duas cargas do mesmo arquivo produziram balances diferentes")
	}
}

func TestShippedBalanceGrowsFishToHarvest(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	state := sim.NewState(1, -180, 0)
	state.Cash = 5_000_000

	actions := []sim.Action{
		{ID: 1, Kind: sim.ActionBuyTank, At: 1, TankKind: sim.TankEarthPond},
		{ID: 2, Kind: sim.ActionStock, At: 2, Tank: 1, Amount: 2_000},
		{ID: 3, Kind: sim.ActionBuyFeed, At: 3, Tank: 1, Amount: 2_000},
		{ID: 4, Kind: sim.ActionBuyUpgrade, At: 4, Tank: 1, Auto: sim.AutoFeeder},
	}

	out, err := sim.Advance(sim.Input{State: state, Until: 120 * sim.TicksPerDay, Balance: &b, Actions: actions})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	for _, o := range out.Outcomes {
		if !o.Applied {
			t.Fatalf("acao %d rejeitada: %v", o.ID, o.Reason)
		}
	}

	grams := out.State.Tanks[0].Batches[0].MeanMass.Grams()
	if grams < 100 || grams > 600 {
		t.Errorf("peso apos 120 dias partindo de alevino = %d g, esperava entre 100 e 600", grams)
	}
	if fish := out.State.Tanks[0].Batches[0].Fish; fish < 1_200 {
		t.Errorf("sobraram %d de 2000 peixes em 120 dias: mortalidade alta demais", fish)
	}
}
