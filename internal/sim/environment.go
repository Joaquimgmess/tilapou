package sim

const (
	halfDayHours = 12
	hoursPerDay  = 24
)

// TemperatureAt returns the water temperature in thousandths of a degree Celsius.
func TemperatureAt(b *Balance, tick Tick, zone ZoneOffset) MilliCelsius {
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

	// ambient e o oxigenio da agua de fora naquela hora: e para ele que a renovacao puxa.
	ambient := int64(b.Water.BaselineOxygen) + mulDivFloor(swing, shape, int64(UnitPPM)) - swing/2

	level := ambient - mulDivFloor(densityMilliKgPerM3(t), int64(b.Water.BiomassDrawPPM), rootScale)

	if t.Aerating {
		level += int64(b.Water.AeratorRecovery)
	}

	// A renovacao recupera o deficit, e para na linha de base: agua nova nao traz mais
	// oxigenio do que a agua tem. Sem o teto, uma renovacao acima de 100%% multiplicaria o
	// deficit e o tanque acabaria com mais oxigenio do que o ambiente.
	renewal := int64(b.Tanks[t.Kind].RenewalPPMPerHour)
	if renewal > 0 {
		if deficit := ambient - level; deficit > 0 {
			level = min(level+mulDivFloor(deficit, renewal, int64(UnitPPM)), ambient)
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

// SeedOxygen writes the equilibrium oxygen of the current tick, in micrograms per litre.
func (s *State) SeedOxygen(b *Balance) {
	for i := range s.TankCount {
		t := &s.Tanks[i]
		t.Oxygen = oxygenAt(b, t, s.Tick, s.Zone)
	}
}
