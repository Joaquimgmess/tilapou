package sim

// ZoneOffset is the farm's time zone, in ticks.
type ZoneOffset int32

// DayPhase is the position of a tick in the local calendar; Day counts from tick zero.
type DayPhase struct {
	Minute int32
	Hour   int32
	Day    int64
}

// At places the tick in the given time zone.
func (t Tick) At(zone ZoneOffset) DayPhase {
	local := t + Tick(zone)
	minuteOfDay := floorMod(int64(local), int64(TicksPerDay))

	return DayPhase{
		Minute: int32(minuteOfDay),
		Hour:   int32(minuteOfDay / 60),
		Day:    floorDiv(int64(local), int64(TicksPerDay)),
	}
}

// WindowStart returns the start of the window containing t, or t if window is not positive.
func (t Tick) WindowStart(window Tick) Tick {
	if window <= 0 {
		return t
	}
	return Tick(floorDiv(int64(t), int64(window)) * int64(window))
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 {
	return a - floorDiv(a, b)*b
}
