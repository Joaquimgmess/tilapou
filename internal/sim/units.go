package sim

// Tick is the time unit of the simulation: one minute of game time.
type Tick int64

// Micrograms is mass in micrograms.
type Micrograms int64

// Coins is money in cents.
type Coins int64

// MilliCelsius is temperature in thousandths of a degree.
type MilliCelsius int32

// MicrogramsPerLiter is dissolved oxygen in micrograms per litre.
type MicrogramsPerLiter int32

// Litres is volume in litres.
type Litres int64

// FishCount counts fish.
type FishCount int32

// PPM is a fraction in parts per million, used in place of floating point.
type PPM int32

// PPM scales and the conversion between tick and day.
const (
	OnePPM         PPM = 1
	UnitPPM        PPM = 1_000_000
	TicksPerDay        = Tick(24 * 60)
	MinutesPerTick     = 1
)

// Mass conversion factors.
const (
	MicrogramsPerGram     Micrograms = 1_000_000
	MicrogramsPerKilogram Micrograms = 1_000 * MicrogramsPerGram
)

// Grams truncates the remainder.
func (m Micrograms) Grams() int64 {
	return int64(m / MicrogramsPerGram)
}

// Apply scales v by the fraction p, rounding down.
func (p PPM) Apply(v int64) int64 {
	return mulDivFloor(v, int64(p), int64(UnitPPM))
}
