package sim

import "math/bits"

// Seed is the seed of the simulation's deterministic generator.
type Seed uint64

// Purpose isolates the randomness stream of each roll consumer.
type Purpose uint16

// Roll purposes: each one generates its own sequence for the same key.
const (
	PurposeUnknown Purpose = iota
	PurposeMortality
	PurposeStarvation
	PurposeDisease
	// Slot queimado: mexer aqui desloca os ordinais e muda o replay de todo save.
	_
	PurposeMarket
	PurposeFeedPrice
	PurposeDiseaseDeath
)

const (
	mixA        = 0xbf58476d1ce4e5b9
	mixB        = 0x94d049bb133111eb
	goldenGamma = 0x9e3779b97f4a7c15
)

// RollKey is the coordinate of a roll; the same key always returns the same value.
type RollKey struct {
	Tick    Tick
	Tank    TankID
	Batch   BatchID
	Purpose Purpose
}

// Roll is deterministic and uniform over the whole uint64.
func (s Seed) Roll(k RollKey) uint64 {
	counter := uint64(k.Tick)*goldenGamma ^
		uint64(k.Tank)<<40 ^
		uint64(k.Batch)<<16 ^
		uint64(k.Purpose)

	return mix64(mix64(uint64(s)^counter) ^ goldenGamma*counter)
}

// RollBelow returns a uniform value in [0, bound), or 0 if bound is not positive.
func (s Seed) RollBelow(k RollKey, bound int64) int64 {
	if bound <= 0 {
		return 0
	}

	high, _ := bits.Mul64(s.Roll(k), uint64(bound))

	return int64(high)
}

// Chance saturates to false below zero and to true from UnitPPM on.
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
