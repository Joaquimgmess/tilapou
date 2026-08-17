package sim

const (
	halfDayHours = 12
	hoursPerDay  = 24
)

func temperatureAt(b *Balance, tick Tick, zone ZoneOffset) MilliCelsius {
	phase := tick.At(zone)
	swing := int64(b.Water.DailyTempSwing)
	shape := triangular(int64(phase.Hour), int64(b.Water.TempPeakHour))

	return MilliCelsius(int64(seasonalTemp(b, tick, zone)) + mulDivFloor(swing, shape, int64(UnitPPM)) - swing/2)
}

func seasonalTemp(b *Balance, tick Tick, zone ZoneOffset) MilliCelsius {
	if b.Water.SeasonDays <= 0 || b.Water.SeasonSwing == 0 {
		return b.Water.BaseTemp
	}

	day := tick.At(zone).Day
	shape := triangular(floorMod(day, b.Water.SeasonDays)*hoursPerDay/b.Water.SeasonDays,
		b.Water.SeasonPeakDay*hoursPerDay/b.Water.SeasonDays)

	swing := int64(b.Water.SeasonSwing)

	return MilliCelsius(int64(b.Water.BaseTemp) + mulDivFloor(swing, shape, int64(UnitPPM)) - swing/2)
}

func oxygenAt(b *Balance, t *Tank, tick Tick, zone ZoneOffset) MicrogramsPerLiter {
	phase := tick.At(zone)
	swing := int64(b.Water.DailySwing)
	shape := triangular(int64(phase.Hour), int64(b.Water.PeakHour))

	level := int64(b.Water.BaselineOxygen) + mulDivFloor(swing, shape, int64(UnitPPM)) - swing/2

	level -= mulDivFloor(densityMilliKgPerM3(t), int64(b.Water.BiomassDrawPPM), rootScale)

	if t.Aerating {
		level += int64(b.Water.AeratorRecovery)
	}

	renewal := int64(b.Tanks[t.Kind].RenewalPPMPerHour)
	if renewal > 0 {
		deficit := int64(b.Water.BaselineOxygen) - level
		if deficit > 0 {
			level += mulDivFloor(deficit, renewal, int64(UnitPPM))
		}
	}

	return MicrogramsPerLiter(max(level, 0))
}

func densityMilliKgPerM3(t *Tank) int64 {
	if t.Litres <= 0 {
		return 0
	}

	return mulDivFloor(int64(t.Biomass()), 1, rootScale*int64(t.Litres))
}

func triangular(hour, peak int64) int64 {
	distance := floorMod(hour-peak, hoursPerDay)
	if distance > halfDayHours {
		distance = hoursPerDay - distance
	}

	return int64(UnitPPM) - mulDivFloor(int64(UnitPPM), distance, halfDayHours)
}

// TemperatureAt devolve a temperatura da agua em milesimos de grau Celsius.
func TemperatureAt(b *Balance, tick Tick, zone ZoneOffset) MilliCelsius {
	return temperatureAt(b, tick, zone)
}

// SeedOxygen grava o oxigenio de equilibrio do tick atual, em microgramas por litro.
func (s *State) SeedOxygen(b *Balance) {
	for i := range s.TankCount {
		t := &s.Tanks[i]
		t.Oxygen = oxygenAt(b, t, s.Tick, s.Zone)
	}
}
