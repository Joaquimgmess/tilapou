package save_test

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/sim"
	"github.com/Joaquimgmess/tilapou/internal/sim/save"
)

func sample() sim.State {
	s := sim.NewState(42, -180, 12_345)
	s.Cash = 987_654
	s.LifetimeEarned = 1_234_567

	id, _ := s.AddTank(sim.TankNetCage, 6_000)
	s.StockTank(id, 900, 250*sim.MicrogramsPerGram, 0)
	s.LoadFeed(id, 300*sim.MicrogramsPerKilogram)

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
		t.Error("estado nao sobreviveu ao round trip")
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
