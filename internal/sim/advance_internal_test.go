package sim

import (
	"errors"
	"slices"
	"testing"
)

func testBalance(t *testing.T) *Balance {
	t.Helper()

	temp, err := NewCurve([]CurvePoint{
		{X: 13_000, Y: 0},
		{X: 18_000, Y: 50_000},
		{X: 22_000, Y: 500_000},
		{X: 25_000, Y: 700_000},
		{X: 27_000, Y: 1_250_000},
		{X: 30_000, Y: 1_400_000},
		{X: 33_000, Y: 700_000},
	})
	if err != nil {
		t.Fatalf("NewCurve() error = %v", err)
	}

	b := &Balance{
		Version: 1,
		Growth: GrowthBalance{
			TGCPPM:         1400,
			ReferenceTemp:  26_000,
			MaxMass:        1_100 * MicrogramsPerGram,
			FingerlingMass: 30 * MicrogramsPerGram,
			HarvestMass:    800 * MicrogramsPerGram,
			TempMultiplier: temp,
		},
		Water: WaterBalance{
			BaseTemp:        27_000,
			DailyTempSwing:  4_000,
			TempPeakHour:    15,
			SeasonSwing:     9_000,
			SeasonDays:      365,
			SeasonPeakDay:   30,
			FeedingMin:      3_000,
			Critical:        2_000,
			Lethal:          1_000,
			PeakHour:        17,
			DailySwing:      6_000,
			BaselineOxygen:  6_000,
			BiomassDrawPPM:  900,
			AeratorRecovery: 2_500,
			AeratorOn:       3_200,
			AeratorOff:      5_200,
		},
		Death: MortalityBalance{
			HypoxiaTicksToLethal: 240,
			HypoxiaRatePPM:       3_000,
			StarvationTicksGrace: 4_320,
			StarvationRatePPM:    1_000,
		},
		Progression: ProgressionBalance{
			CostFactorPPM:    1_150_000,
			PrestigeDivisor:  1_000_000,
			PrestigeBonusPPM: 50_000,
			ContractBonusPPM: 100_000,
			RestartCash:      50_000,
			RestartFish:      2_000,
			RestartFeed:      200 * MicrogramsPerKilogram,
		},
		Economy: EconomyBalance{
			FingerlingPrice: 80,
			AeratorCostTick: 2,
		},
		Market: MarketBalance{
			FishBasePerKg:  900,
			FeedBasePerKg:  320,
			SwingPPM:       220_000,
			FeedSwingPPM:   140_000,
			PeriodTicks:    21 * TicksPerDay,
			Seed:           20260815,
			ViableRatioPPM: 1_250_000,
		},
		Credit: CreditBalance{
			MaxPrincipal: 1_500_000,
			DailyRatePPM: 4_500,
		},
		Shock: ShockBalance{
			CheckEvery:     3 * TicksPerDay,
			TreatmentCost:  90_000,
			CarrierTicks:   120 * TicksPerDay,
			CarrierRiskPPM: 2_600_000,
		},
	}

	b.Ration.Steps = [maxRationSteps]RationStep{
		{UpToMass: 5 * MicrogramsPerGram, RatePPMDay: 250_000, MealsPerDay: 6},
		{UpToMass: 30 * MicrogramsPerGram, RatePPMDay: 100_000, MealsPerDay: 5},
		{UpToMass: 175 * MicrogramsPerGram, RatePPMDay: 40_000, MealsPerDay: 4},
		{UpToMass: 350 * MicrogramsPerGram, RatePPMDay: 30_000, MealsPerDay: 3},
		{UpToMass: 1_200 * MicrogramsPerGram, RatePPMDay: 20_000, MealsPerDay: 2},
	}
	b.Ration.Len = 5
	b.Ration.TargetFCRPPM = 1_600_000
	b.Ration.MaintenancePPM = 3_500

	b.Tanks = [tankKindCount]TankSpec{
		TankEarthPond:     {MaxDensityPerM3: 5, BaseCost: 300_000, UpkeepPerDay: 900, Litres: 1_000_000},
		TankNetCage:       {MaxDensityPerM3: 160, RenewalPPMPerHour: 900_000, BaseCost: 800_000, UpkeepPerDay: 1_800, Litres: 6_000},
		TankBiofloc:       {MaxDensityPerM3: 80, RenewalPPMPerHour: 200_000, BaseCost: 3_500_000, UpkeepPerDay: 5_200, Litres: 30_000},
		TankRecirculation: {MaxDensityPerM3: 120, RenewalPPMPerHour: 990_000, BaseCost: 12_000_000, UpkeepPerDay: 14_000, Litres: 20_000},
	}

	b.Market.Classes = [maxPriceClasses]PriceClass{
		{UpToMass: 400 * MicrogramsPerGram, PPM: 720_000},
		{UpToMass: 600 * MicrogramsPerGram, PPM: 880_000},
		{UpToMass: 900 * MicrogramsPerGram, PPM: UnitPPM},
		{UpToMass: 1_100 * MicrogramsPerGram, PPM: 1_120_000},
		{UpToMass: 2_000 * MicrogramsPerGram, PPM: 940_000},
	}
	b.Market.ClassCount = 5

	b.Shock.Diseases = [maxDiseases]DiseaseSpec{
		{MinTemp: 28_500, MaxTemp: 40_000, OutbreakPPM: 90_000, DeathPPM: 41, Ticks: 5 * int32(TicksPerDay)},
		{MinTemp: 0, MaxTemp: 24_000, OutbreakPPM: 70_000, DeathPPM: 27, Ticks: 8 * int32(TicksPerDay)},
	}
	b.Shock.DiseaseCount = 2

	b.Automation = [autoKindCount]AutomationSpec{
		AutoFeeder:     {Cost: 250_000},
		AutoAerator:    {Cost: 600_000},
		AutoHarvester:  {Cost: 2_000_000},
		AutoTechnician: {Cost: 7_500_000},
		AutoContract:   {Cost: 25_000_000},
	}

	b.Shock.CheckEvery = 0

	if err := b.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	return b
}

func stockedFarm(t *testing.T, seed Seed) State {
	t.Helper()

	s := NewState(seed, -180, 0)
	s.Cash = 500_000

	id, ok := s.addTank(TankEarthPond, 1_000_000)
	if !ok {
		t.Fatal("addTank() falhou numa fazenda vazia")
	}

	tank := s.tank(id)
	if !tank.addBatch(s.NextBatchID, 2_000, 300*MicrogramsPerGram, 0) {
		t.Fatal("addBatch() falhou num tanque vazio")
	}
	s.NextBatchID++
	tank.FeedStock = 400 * MicrogramsPerKilogram
	tank.ServedUntil = Tick(maxInt32)

	return s
}

func TestAdvanceToCurrentTickIsIdentity(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	start := stockedFarm(t, 7)

	out, err := Advance(Input{State: start, Until: start.Tick, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if out.State != start {
		t.Error("avancar para o proprio tick alterou o estado")
	}
	if len(out.Events) != 0 {
		t.Errorf("emitiu %d eventos sem avancar o tempo", len(out.Events))
	}
}

func TestAdvanceDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	start := stockedFarm(t, 11)
	before := start

	if _, err := Advance(Input{State: start, Until: 5_000, Balance: b}); err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if start != before {
		t.Error("Advance mutou o estado recebido")
	}
}

func TestAdvanceRejectsBadInput(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	start := stockedFarm(t, 3)

	tests := map[string]struct {
		in   Input
		want error
	}{
		"sem balance":      {Input{State: start, Until: 10}, ErrNoBalance},
		"tempo pra tras":   {Input{State: start, Until: -1, Balance: b}, ErrBackwardsTime},
		"acoes bagunçadas": {Input{State: start, Until: 10, Balance: b, Actions: []Action{{At: 5}, {At: 2}}}, ErrActionsUnsorted},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Advance(tt.in); !errors.Is(err, tt.want) {
				t.Errorf("Advance() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDeferredActionsAreNeverConsumed(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	start := stockedFarm(t, 5)

	actions := []Action{{ID: 1, Kind: ActionAerate, At: 5_000, Tank: 1, Amount: 1}}

	out, err := Advance(Input{State: start, Until: 100, Balance: b, Actions: actions})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if len(out.Outcomes) != 0 {
		t.Fatalf("acao do futuro foi consumida: %+v", out.Outcomes)
	}
	if out.State.Tanks[0].Aerating {
		t.Error("acao do futuro teve efeito no presente")
	}
}

func TestRejectedActionIsVisibleAndLeavesCashAlone(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	start := stockedFarm(t, 9)
	start.Cash = 10

	actions := []Action{{ID: 42, Kind: ActionBuyTank, At: 3, TankKind: TankRecirculation}}

	out, err := Advance(Input{State: start, Until: 10, Balance: b, Actions: actions})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if len(out.Outcomes) != 1 || out.Outcomes[0].Applied {
		t.Fatalf("outcomes = %+v, queria uma rejeicao", out.Outcomes)
	}
	if out.Outcomes[0].Reason != RejectNotEnoughCash {
		t.Errorf("reason = %v, want %v", out.Outcomes[0].Reason, RejectNotEnoughCash)
	}
	if out.State.TankCount != 1 {
		t.Errorf("a rejeicao criou tanque: %d tanques", out.State.TankCount)
	}
	if out.State.Cash > 10 {
		t.Errorf("caixa = %d, a rejeicao nao pode creditar", out.State.Cash)
	}

	rejected := slices.ContainsFunc(out.Events, func(e Event) bool {
		return e.Kind == EventActionRejected
	})
	if !rejected {
		t.Error("rejeicao nao apareceu no feed de eventos")
	}
}

func FuzzAdvanceIsPartitionInvariant(f *testing.F) {
	f.Add(uint64(1), int64(2_000), int64(700))
	f.Add(uint64(99), int64(10_000), int64(1))
	f.Add(uint64(7), int64(4_321), int64(4_320))
	f.Add(uint64(3), int64(60), int64(30))

	f.Fuzz(func(t *testing.T, rawSeed uint64, total, split int64) {
		if total <= 0 || total > 200_000 {
			t.Skip()
		}
		if split < 0 || split > total {
			t.Skip()
		}

		b := testBalance(t)
		start := stockedFarm(t, Seed(rawSeed))

		whole, err := Advance(Input{State: start, Until: Tick(total), Balance: b})
		if err != nil {
			t.Fatalf("Advance inteiro: %v", err)
		}

		first, err := Advance(Input{State: start, Until: Tick(split), Balance: b})
		if err != nil {
			t.Fatalf("Advance primeira parte: %v", err)
		}
		second, err := Advance(Input{State: first.State, Until: Tick(total), Balance: b})
		if err != nil {
			t.Fatalf("Advance segunda parte: %v", err)
		}

		if whole.State != second.State {
			t.Fatalf("estado divergiu ao particionar em %d de %d ticks", split, total)
		}

		joined := slices.Concat(first.Events, second.Events)
		if !slices.Equal(whole.Events, joined) {
			t.Fatalf("eventos divergiram: inteiro=%d particionado=%d", len(whole.Events), len(joined))
		}
	})
}
