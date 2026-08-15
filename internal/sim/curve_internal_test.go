package sim

import (
	"errors"
	"testing"
)

func growthCurve(t *testing.T) Curve {
	t.Helper()

	c, err := NewCurve([]CurvePoint{
		{X: 13_000, Y: 0},
		{X: 18_000, Y: 50_000},
		{X: 22_000, Y: 500_000},
		{X: 25_000, Y: 700_000},
		{X: 27_000, Y: 1_250_000},
		{X: 30_000, Y: 1_400_000},
		{X: 33_000, Y: 700_000},
	})
	if err != nil {
		t.Fatalf("NewCurve() error = %v", err)
	}

	return c
}

func TestCurveAtHitsAnchorsAndClamps(t *testing.T) {
	t.Parallel()

	c := growthCurve(t)

	tests := map[string]struct {
		x    int64
		want int64
	}{
		"abaixo do primeiro": {0, 0},
		"primeiro ancora":    {13_000, 0},
		"ancora do meio":     {25_000, 700_000},
		"ultimo ancora":      {33_000, 700_000},
		"acima do ultimo":    {40_000, 700_000},
		"interpolado":        {26_000, 975_000},
		"queda por calor":    {31_500, 1_050_000},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := c.At(tt.x); got != tt.want {
				t.Errorf("At(%d) = %d, want %d", tt.x, got, tt.want)
			}
		})
	}
}

func TestCurveAtIsMonotoneBetweenAnchors(t *testing.T) {
	t.Parallel()

	c := growthCurve(t)

	previous := c.At(13_000)
	for x := int64(13_000); x <= 30_000; x += 25 {
		got := c.At(x)
		if got < previous {
			t.Fatalf("At(%d) = %d caiu abaixo de %d na faixa crescente", x, got, previous)
		}
		previous = got
	}
}

func TestNewCurveRejectsBadInput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		points []CurvePoint
		want   error
	}{
		"vazia":         {nil, ErrCurveEmpty},
		"fora de ordem": {[]CurvePoint{{X: 10}, {X: 5}}, ErrCurveNotSorted},
		"x repetido":    {[]CurvePoint{{X: 10}, {X: 10}}, ErrCurveNotSorted},
		"longa demais":  {make([]CurvePoint, maxCurvePoints+1), ErrCurveTooLong},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewCurve(tt.points); !errors.Is(err, tt.want) {
				t.Errorf("NewCurve() error = %v, want %v", err, tt.want)
			}
		})
	}
}
