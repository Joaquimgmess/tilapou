package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

func sizedSnapshot() client.Snapshot {
	return client.Snapshot{
		FarmID: "x", CashCents: 1_234_567, Debt: 500_000, InterestDay: 2_250, RunwayDays: 18,
		Fish: 3_400, Tick: 47 * 24 * 60, Hour: 4, TempMilliC: 26_500,
		Prices: client.Prices{FishKgCents: 942, FeedKgCents: 318, RatioPPM: 1_850_000, ViablePPM: 1_250_000},
		Series: client.Series{
			FishKgCents: []int64{900, 910, 880, 940, 960, 942, 930, 915, 950, 970, 942},
			FeedKgCents: []int64{320, 318, 322, 315, 310, 318, 325, 319, 316, 312, 318},
		},
		Tanks: []client.Tank{
			{
				ID: 1, Kind: "viveiro_escavado", Fish: 2_000, MeanGrams: 306, FeedKg: 180,
				OxygenUgL: 5_400, DensityMilli: 600, BatchID: 3, Capacity: 5_000, ServedFor: 240,
				BreakEven: 967, StockAdvice: 1_240, CostPerFish: 599,
				PriceKgCents: 678, ValueCents: 414_936, CostCents: 393_600, MarginCents: 21_336,
				CostPerKg: 674, ClassPPM: 720_000, NextClassG: 600, NextClassGain: 220_000,
				Decision: client.Decision{
					SellNowCents: 414_936, SellNowMargin: 21_336, HoldToGrams: 600, HoldDays: 62,
					HoldCents: 994_800, HoldMargin: 167_440, HoldFeedCents: 574_560, HoldReached: true,
					BreakEvenPerKg: 674, GainPerDayMg: 2_800, FeedPerDayG: 17_100,
					FeedCostPerDay: 5_472, DaysOfFeed: 4,
				},
				Upgrades: []client.Upgrade{{Kind: "comedouro", CostCents: 80_000}},
			},
			{
				ID: 2, Kind: "tanque_rede", Fish: 1_400, MeanGrams: 612, FeedKg: 0,
				OxygenUgL: 1_900, BatchID: 4, Capacity: 960, PriceKgCents: 829,
				BreakEven: 1_820, StockAdvice: 0, CostPerFish: 599,
				ValueCents: 754_120, MarginCents: 298_004, ClassPPM: 880_000, NextClassG: 900,
				NextClassGain: 140_000,
			},
		},
	}
}

func everyUpgrade() []client.Upgrade {
	return []client.Upgrade{
		{Kind: "comedouro", CostCents: 80_000}, {Kind: "aerador", CostCents: 180_000},
		{Kind: "peao", CostCents: 800_000}, {Kind: "tecnico", CostCents: 1_600_000},
		{Kind: "contrato", CostCents: 3_200_000},
	}
}

func TestFrameFitsCommonTerminals(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Tanks[0].Upgrades = everyUpgrade()

	overlays := map[string]*menu{
		"painel":         nil,
		"menu do tanque": tankMenu(snap, snap.Tanks[0]),
		"menu do galpao": shedMenu(snap, snap.Tanks[0]),
	}

	for _, size := range []struct{ w, h int }{{120, 40}, {100, 35}, {80, 24}} {
		for name, overlay := range overlays {
			m := New(nil)
			m.snapshot = snap
			m.menu = overlay
			m.width, m.height = size.w, size.h

			lines := strings.Split(strings.TrimRight(m.render(), "\n"), "\n")

			widest := 0
			for _, line := range lines {
				widest = max(widest, lipgloss.Width(line))
			}

			if widest > size.w || len(lines) > size.h {
				t.Errorf("%s em %dx%d saiu %dx%d", name, size.w, size.h, widest, len(lines))
			}
		}
	}
}

func TestTinyTerminalExplainsTheDeficit(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 60, 15

	frame := m.render()
	for _, want := range []string{"pequeno demais", "faltam"} {
		if !strings.Contains(frame, want) {
			t.Errorf("a mensagem de terminal pequeno nao diz %q:\n%s", want, frame)
		}
	}
}
