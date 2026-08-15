package sim

import "math/bits"

type Seed uint64

type Purpose uint16

const (
	PurposeUnknown Purpose = iota
	PurposeMortality
	PurposeDisease
	PurposeWeather
	PurposeMarket
	PurposeEvent
)

const (
	mixA        = 0xbf58476d1ce4e5b9
	mixB        = 0x94d049bb133111eb
	goldenGamma = 0x9e3779b97f4a7c15
)

type RollKey struct {
	Tick    Tick
	Tank    TankID
	Batch   BatchID
	Purpose Purpose
}

func (s Seed) Roll(k RollKey) uint64 {
	counter := uint64(k.Tick)*goldenGamma ^
		uint64(k.Tank)<<40 ^
		uint64(k.Batch)<<16 ^
		uint64(k.Purpose)

	return mix64(mix64(uint64(s)^counter) ^ goldenGamma*counter)
}

func (s Seed) RollBelow(k RollKey, bound int64) int64 {
	if bound <= 0 {
		return 0
	}

	high, _ := bits.Mul64(s.Roll(k), uint64(bound))

	return int64(high)
}

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
