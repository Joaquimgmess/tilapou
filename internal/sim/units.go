package sim

// Tick e a unidade de tempo da simulacao: um minuto de jogo.
type Tick int64

// Micrograms e massa em microgramas.
type Micrograms int64

// Coins e dinheiro em centavos.
type Coins int64

// MilliCelsius e temperatura em milesimos de grau.
type MilliCelsius int32

// MicrogramsPerLiter e oxigenio dissolvido em microgramas por litro.
type MicrogramsPerLiter int32

// Litres e volume em litros.
type Litres int64

// FishCount conta peixes.
type FishCount int32

// PPM e uma fracao em partes por milhao, usada no lugar de ponto flutuante.
type PPM int32

// Escalas de PPM e a conversao entre tick e dia.
const (
	OnePPM         PPM = 1
	UnitPPM        PPM = 1_000_000
	TicksPerDay        = Tick(24 * 60)
	MinutesPerTick     = 1
)

// Fatores de conversao de massa.
const (
	MicrogramsPerGram     Micrograms = 1_000_000
	MicrogramsPerKilogram Micrograms = 1_000 * MicrogramsPerGram
)

// Grams trunca o resto.
func (m Micrograms) Grams() int64 {
	return int64(m / MicrogramsPerGram)
}

// Apply escala v pela fracao p, arredondando para baixo.
func (p PPM) Apply(v int64) int64 {
	return mulDivFloor(v, int64(p), int64(UnitPPM))
}
