package sim

type Tick int64

type Micrograms int64

type Coins int64

type MilliCelsius int32

type MicrogramsPerLiter int32

type Litres int64

type FishCount int32

type PPM int32

const (
	OnePPM         PPM = 1
	UnitPPM        PPM = 1_000_000
	TicksPerDay        = Tick(24 * 60)
	MinutesPerTick     = 1
)

const (
	MicrogramsPerGram     Micrograms = 1_000_000
	MicrogramsPerKilogram Micrograms = 1_000 * MicrogramsPerGram
)

func (m Micrograms) Grams() int64 {
	return int64(m / MicrogramsPerGram)
}

func (p PPM) Apply(v int64) int64 {
	return mulDivFloor(v, int64(p), int64(UnitPPM))
}
