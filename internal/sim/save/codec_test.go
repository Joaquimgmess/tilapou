package save_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/sim"
	"github.com/Joaquimgmess/tilapou/internal/sim/save"
)

func sample() sim.State {
	s := sim.NewState(42, -180, 12_345)
	s.Cash = 987_654
	s.LifetimeEarned = 1_234_567
	s.Prestige = 3
	s.EventSeq = 91
	s.Debt = 45_000
	s.DebtCarry = 17
	s.LastCycle = sim.Cycle{
		Fish: 1_234, Mass: 500 * sim.MicrogramsPerKilogram, Revenue: 9_999,
		Cost: 111, CostPerKg: 7, PricePerKg: 9, FCRPPM: 1_500_000,
	}

	id, _ := s.AddTank(sim.TankNetCage, 6_000)
	s.StockTank(id, 900, 250*sim.MicrogramsPerGram, 5_000)
	s.LoadFeed(id, 300*sim.MicrogramsPerKilogram, 317)

	tank := &s.Tanks[0]
	tank.ServedUntil = 12_400
	tank.Upgrades = 1<<sim.AutoFeeder | 1<<sim.AutoAerator
	tank.Oxygen = 5_100
	tank.Aerating = true
	tank.FeedCarry = 31
	tank.UpkeepCarry = 47
	tank.CarrierUntil = 13_000
	tank.Accrual = sim.Accrual{
		Window: 12_300, HypoxiaDeaths: 3, StarvationDeaths: 5, DiseaseDeaths: 7,
		FeedEaten: 88 * sim.MicrogramsPerKilogram, MassGained: 99 * sim.MicrogramsPerKilogram,
	}

	batch := &tank.Batches[0]
	batch.GrowthCarry = 61
	batch.FeedEaten = 44 * sim.MicrogramsPerKilogram
	batch.MassGained = 55 * sim.MicrogramsPerKilogram
	batch.CostCarry = 73
	batch.Sick = 9
	batch.HypoxiaTicks = 11
	batch.StarvationTicks = 13

	return s
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()

	want := sample()

	raw, err := save.Encode(want)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got, err := save.Decode(raw)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if got != want {
		t.Errorf("estado nao sobreviveu ao round trip\n  got  %+v\n  want %+v", got, want)
	}
}

func TestDecodeRejectsUnknownVersion(t *testing.T) {
	t.Parallel()

	if _, err := save.Decode([]byte(`{"version":999}`)); err == nil {
		t.Error("Decode() aceitou uma versao desconhecida")
	}
}

func TestEncodeSkipsEmptySlots(t *testing.T) {
	t.Parallel()

	raw, err := save.Encode(sample())
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if len(raw) > 2_000 {
		t.Errorf("save com um tanque ocupou %d bytes: os slots vazios estao indo junto", len(raw))
	}
}

func leafZeros(v reflect.Value, path string, found *[]string) {
	if v.Kind() == reflect.Struct {
		for i := range v.NumField() {
			if field := v.Type().Field(i); field.IsExported() {
				leafZeros(v.Field(i), path+"."+field.Name, found)
			}
		}

		return
	}

	if v.Kind() == reflect.Array {
		if v.Len() > 0 {
			leafZeros(v.Index(0), path+"[0]", found)
		}

		return
	}

	if v.IsZero() {
		*found = append(*found, path)
	}
}

func TestSampleTouchesEveryFieldSoTheRoundTripCanSeeThem(t *testing.T) {
	t.Parallel()

	var zeroed []string
	leafZeros(reflect.ValueOf(sample()), "State", &zeroed)

	if len(zeroed) > 0 {
		t.Errorf("o round trip e cego para %d campos porque o sample() os deixa zerados:\n  %s",
			len(zeroed), strings.Join(zeroed, "\n  "))
	}
}
