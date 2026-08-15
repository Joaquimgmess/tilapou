package sim

import (
	"math"
	"math/big"
	"testing"
)

func TestAddSatSaturatesAtBoundary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		a, b, want int64
	}{
		"soma comum":        {2, 3, 5},
		"estouro positivo":  {math.MaxInt64, 1, math.MaxInt64},
		"estouro negativo":  {math.MinInt64, -1, math.MinInt64},
		"limite positivo":   {math.MaxInt64, math.MaxInt64, math.MaxInt64},
		"limite negativo":   {math.MinInt64, math.MinInt64, math.MinInt64},
		"sinais opostos":    {math.MaxInt64, math.MinInt64, -1},
		"zero e o neutro":   {0, math.MinInt64, math.MinInt64},
		"subtracao no zero": {0, 0, 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := addSat(tt.a, tt.b); got != tt.want {
				t.Errorf("addSat(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSubSatSaturatesAtBoundary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		a, b, want int64
	}{
		"comum":            {10, 3, 7},
		"estouro negativo": {math.MinInt64, 1, math.MinInt64},
		"minInt64 como b":  {0, math.MinInt64, math.MaxInt64},
		"positivo grande":  {math.MaxInt64, math.MinInt64, math.MaxInt64},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := subSat(tt.a, tt.b); got != tt.want {
				t.Errorf("subSat(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMulDivRoundsTowardTheRightSide(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		v, n, d          int64
		wantFloor, wantC int64
	}{
		"exato":              {100, 3, 3, 100, 100},
		"positivo com resto": {10, 1, 3, 3, 4},
		"negativo com resto": {-10, 1, 3, -4, -3},
		"divisor maior":      {1, 1, 1000, 0, 1},
		"produto estoura":    {math.MaxInt64, 2, 1, math.MaxInt64, math.MaxInt64},
		"divisao por zero":   {10, 1, 0, math.MaxInt64, math.MaxInt64},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := mulDivFloor(tt.v, tt.n, tt.d); got != tt.wantFloor {
				t.Errorf("mulDivFloor(%d, %d, %d) = %d, want %d", tt.v, tt.n, tt.d, got, tt.wantFloor)
			}
			if got := mulDivCeil(tt.v, tt.n, tt.d); got != tt.wantC {
				t.Errorf("mulDivCeil(%d, %d, %d) = %d, want %d", tt.v, tt.n, tt.d, got, tt.wantC)
			}
		})
	}
}

func FuzzMulDivFloorMatchesBigInt(f *testing.F) {
	f.Add(int64(10), int64(3), int64(7))
	f.Add(int64(-10), int64(3), int64(7))
	f.Add(int64(math.MaxInt64), int64(1), int64(1))
	f.Add(int64(1), int64(math.MaxInt64), int64(math.MaxInt64))

	f.Fuzz(func(t *testing.T, v, n, d int64) {
		if d == 0 {
			t.Skip()
		}

		got := mulDivFloor(v, n, d)

		exact := new(big.Int).Mul(big.NewInt(v), big.NewInt(n))
		want := new(big.Int)
		want.Div(exact, big.NewInt(d))

		if !want.IsInt64() {
			if got != math.MaxInt64 && got != math.MinInt64 {
				t.Fatalf("mulDivFloor(%d, %d, %d) = %d, want saturation", v, n, d, got)
			}
			return
		}

		if got != want.Int64() {
			t.Fatalf("mulDivFloor(%d, %d, %d) = %d, want %d", v, n, d, got, want.Int64())
		}
	})
}

func TestCarryTakePartitionsExactly(t *testing.T) {
	t.Parallel()

	const (
		total       = int64(1000)
		numerator   = int64(1)
		denominator = int64(3)
		steps       = 300
	)

	var carry, accumulated int64
	for range steps {
		accumulated = addSat(accumulated, carryTake(total, numerator, denominator, &carry))
	}

	want := total * numerator * steps / denominator
	if accumulated != want {
		t.Errorf("soma com carry = %d, want %d (perda de %d)", accumulated, want, want-accumulated)
	}
}

func TestCarryTakeWithoutCarryLosesValue(t *testing.T) {
	t.Parallel()

	var carry, withCarry, without int64
	for range 300 {
		withCarry = addSat(withCarry, carryTake(1000, 1, 3, &carry))
		without = addSat(without, mulDivFloor(1000, 1, 3))
	}

	if withCarry <= without {
		t.Errorf("carry nao recuperou nada: com=%d sem=%d", withCarry, without)
	}
}

func TestIcbrtIsExact(t *testing.T) {
	t.Parallel()

	for _, v := range []int64{0, 1, 7, 8, 9, 26, 27, 28, 999, 1000, 1001, 1_000_000, 800_000_000_000, math.MaxInt64} {
		got := icbrt(v)
		if got*got*got > v {
			t.Errorf("icbrt(%d) = %d, cubo passou do valor", v, got)
		}
		if got < 2_097_151 && (got+1)*(got+1)*(got+1) <= v {
			t.Errorf("icbrt(%d) = %d, poderia ser maior", v, got)
		}
	}
}

func TestIsqrtIsExact(t *testing.T) {
	t.Parallel()

	for _, v := range []int64{0, 1, 2, 3, 4, 99, 100, 101, 1_000_000, math.MaxInt64} {
		got := isqrt(v)
		if got*got > v {
			t.Errorf("isqrt(%d) = %d, quadrado passou do valor", v, got)
		}
		if got < 3_037_000_499 && (got+1)*(got+1) <= v {
			t.Errorf("isqrt(%d) = %d, poderia ser maior", v, got)
		}
	}
}
