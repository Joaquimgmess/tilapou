package sim

import "math/bits"

// Seed e a semente do gerador deterministico da simulacao.
type Seed uint64

// Purpose isola o fluxo de aleatoriedade de cada consumidor de sorteio.
type Purpose uint16

// Propositos de sorteio: cada um gera sua propria sequencia para a mesma chave.
const (
	PurposeUnknown Purpose = iota
	PurposeMortality
	PurposeStarvation
	PurposeDisease
	PurposeWeather
	PurposeMarket
	PurposeEvent
	PurposeDiseaseDeath
)

const (
	mixA        = 0xbf58476d1ce4e5b9
	mixB        = 0x94d049bb133111eb
	goldenGamma = 0x9e3779b97f4a7c15
)

// RollKey e a coordenada de um sorteio; a mesma chave sempre devolve o mesmo valor.
type RollKey struct {
	Tick    Tick
	Tank    TankID
	Batch   BatchID
	Purpose Purpose
}

// Roll e deterministico e uniforme em todo o uint64.
func (s Seed) Roll(k RollKey) uint64 {
	counter := uint64(k.Tick)*goldenGamma ^
		uint64(k.Tank)<<40 ^
		uint64(k.Batch)<<16 ^
		uint64(k.Purpose)

	return mix64(mix64(uint64(s)^counter) ^ goldenGamma*counter)
}

// RollBelow devolve um valor uniforme em [0, bound), ou 0 se bound nao for positivo.
func (s Seed) RollBelow(k RollKey, bound int64) int64 {
	if bound <= 0 {
		return 0
	}

	high, _ := bits.Mul64(s.Roll(k), uint64(bound))

	return int64(high)
}

// Chance satura em false abaixo de zero e em true a partir de UnitPPM.
func (s Seed) Chance(k RollKey, probability PPM) bool {
	if probability <= 0 {
		return false
	}
	if probability >= UnitPPM {
		return true
	}

	return s.RollBelow(k, int64(UnitPPM)) < int64(probability)
}

func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= mixA
	x ^= x >> 27
	x *= mixB
	x ^= x >> 31

	return x
}
