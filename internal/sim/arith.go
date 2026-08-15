package sim

import "math/bits"

const (
	maxInt64 = int64(^uint64(0) >> 1)
	minInt64 = -maxInt64 - 1
	maxIsqrt = int64(3_037_000_499)
	maxIcbrt = int64(2_097_151)
)

func addSat(a, b int64) int64 {
	sum := a + b
	if (a > 0 && b > 0 && sum < 0) || (a < 0 && b < 0 && sum >= 0) {
		if a > 0 {
			return maxInt64
		}
		return minInt64
	}
	return sum
}

func subSat(a, b int64) int64 {
	if b == minInt64 {
		if a >= 0 {
			return maxInt64
		}
		return addSat(a, maxInt64)
	}
	return addSat(a, -b)
}

func mulDivFloor(value, numerator, denominator int64) int64 {
	q, r, ok := mulDiv128(value, numerator, denominator)
	if !ok {
		return saturationOf(value, numerator, denominator)
	}
	if r != 0 && (value < 0) != (numerator < 0) {
		return q - 1
	}
	return q
}

func mulDivCeil(value, numerator, denominator int64) int64 {
	q, r, ok := mulDiv128(value, numerator, denominator)
	if !ok {
		return saturationOf(value, numerator, denominator)
	}
	if r != 0 && (value < 0) == (numerator < 0) {
		return q + 1
	}
	return q
}

func mulDiv128(value, numerator, denominator int64) (quotient, remainder int64, ok bool) {
	if denominator == 0 {
		return 0, 0, false
	}

	negative := (value < 0) != (numerator < 0) != (denominator < 0)
	a, b, d := absU(value), absU(numerator), absU(denominator)

	hi, lo := bits.Mul64(a, b)
	if hi >= d {
		return 0, 0, false
	}

	qu, ru := bits.Div64(hi, lo, d)
	if qu > uint64(maxInt64) {
		return 0, 0, false
	}

	q, r := int64(qu), int64(ru)
	if negative {
		q, r = -q, -r
	}

	return q, r, true
}

func saturationOf(value, numerator, denominator int64) int64 {
	negative := (value < 0) != (numerator < 0) != (denominator < 0)
	if negative {
		return minInt64
	}
	return maxInt64
}

func absU(v int64) uint64 {
	if v < 0 {
		return uint64(-(v + 1)) + 1
	}
	return uint64(v)
}

func carryTake(total, numerator, denominator int64, carry *int64) int64 {
	scaled, remainder, ok := mulDiv128(total, numerator, denominator)
	if !ok {
		*carry = 0
		return saturationOf(total, numerator, denominator)
	}

	pending := addSat(*carry, remainder)
	extra := pending / denominator
	*carry = pending - extra*denominator

	return addSat(scaled, extra)
}

func isqrt(v int64) int64 {
	if v <= 0 {
		return 0
	}

	x := min(int64(1)<<((bits.Len64(uint64(v))+1)/2), maxIsqrt)
	for {
		next := (x + v/x) / 2
		if next >= x {
			break
		}
		x = next
	}

	for x < maxIsqrt && (x+1)*(x+1) <= v {
		x++
	}

	return x
}

func icbrt(v int64) int64 {
	if v <= 0 {
		return 0
	}

	x := min(int64(1)<<((bits.Len64(uint64(v))+2)/3), maxIcbrt)
	for {
		next := (2*x + v/(x*x)) / 3
		if next >= x {
			break
		}
		x = next
	}

	for x < maxIcbrt && (x+1)*(x+1)*(x+1) <= v {
		x++
	}
	for x > 0 && x*x*x > v {
		x--
	}

	return x
}
