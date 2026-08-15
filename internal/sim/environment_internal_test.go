package sim

import "testing"

func TestSeasonSwingsAcrossTheYear(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	coldest, hottest := MilliCelsius(1<<30), MilliCelsius(-(1 << 30))
	for day := range int64(365) {
		temp := seasonalTemp(b, Tick(day)*TicksPerDay, 0)
		coldest = min(coldest, temp)
		hottest = max(hottest, temp)
	}

	swing := hottest - coldest
	if swing < b.Water.SeasonSwing*8/10 {
		t.Errorf("amplitude anual = %d milli-C, o balance pede perto de %d: a sazonalidade nao esta rodando",
			swing, b.Water.SeasonSwing)
	}
}

func TestSeasonReachesBothDiseaseBands(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	var summer, winter bool
	for day := range int64(365) {
		temp := seasonalTemp(b, Tick(day)*TicksPerDay, 0)
		if _, index, ok := diseaseFor(b, temp); ok {
			switch index {
			case 0:
				summer = true
			case 1:
				winter = true
			}
		}
	}

	if !summer || !winter {
		t.Errorf("o ano nao cruza as duas faixas de doenca: verao=%v inverno=%v", summer, winter)
	}
}
