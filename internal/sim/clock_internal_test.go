package sim

import "testing"

func TestTickAtHandlesNegativeZoneAndTick(t *testing.T) {
	t.Parallel()

	const brasilia = ZoneOffset(-3 * 60)

	tests := map[string]struct {
		tick     Tick
		zone     ZoneOffset
		wantHour int32
		wantDay  int64
	}{
		"epoca":                {0, 0, 0, 0},
		"meio-dia":             {12 * 60, 0, 12, 0},
		"virada":               {24 * 60, 0, 0, 1},
		"zona negativa":        {0, brasilia, 21, -1},
		"tick negativo":        {-1, 0, 23, -1},
		"negativo com zona":    {-60, brasilia, 20, -1},
		"madrugada de risco":   {5 * 60, 0, 5, 0},
		"tres dias de offline": {3*24*60 + 6*60, 0, 6, 3},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tt.tick.At(tt.zone)
			if got.Hour != tt.wantHour {
				t.Errorf("Hour = %d, want %d", got.Hour, tt.wantHour)
			}
			if got.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", got.Day, tt.wantDay)
			}
		})
	}
}

func TestTickAtMinuteIsAlwaysInsideTheDay(t *testing.T) {
	t.Parallel()

	for tick := Tick(-3000); tick < 3000; tick++ {
		got := tick.At(-180)
		if got.Minute < 0 || int64(got.Minute) >= int64(TicksPerDay) {
			t.Fatalf("tick %d rendeu minuto %d fora do dia", tick, got.Minute)
		}
	}
}

func TestWindowStartAlignsToAbsoluteTick(t *testing.T) {
	t.Parallel()

	const window = Tick(60)

	tests := map[string]struct {
		tick Tick
		want Tick
	}{
		"inicio":     {0, 0},
		"meio":       {37, 0},
		"fronteira":  {60, 60},
		"negativo":   {-1, -60},
		"negativo 2": {-61, -120},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tt.tick.WindowStart(window); got != tt.want {
				t.Errorf("WindowStart(%d) = %d, want %d", tt.tick, got, tt.want)
			}
		})
	}
}
