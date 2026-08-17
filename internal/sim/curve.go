package sim

const maxCurvePoints = 12

// CurvePoint is an anchor pair of a Curve.
type CurvePoint struct {
	X int64
	Y int64
}

// Curve is piecewise linear, with up to maxCurvePoints points sorted by X.
type Curve struct {
	Points [maxCurvePoints]CurvePoint
	Len    int32
}

// NewCurve fails with ErrCurveEmpty when there are no points, ErrCurveTooLong above the limit and ErrCurveNotSorted if X does not increase.
func NewCurve(points []CurvePoint) (Curve, error) {
	if len(points) == 0 {
		return Curve{}, ErrCurveEmpty
	}
	if len(points) > maxCurvePoints {
		return Curve{}, ErrCurveTooLong
	}

	var (
		c        Curve
		previous CurvePoint
	)
	for i, p := range points {
		if i > 0 && p.X <= previous.X {
			return Curve{}, ErrCurveNotSorted
		}
		c.Points[i] = p
		previous = p
	}
	c.Len = int32(len(points))

	return c, nil
}

// At interpolates Y at x, clamping at the ends; it returns 0 on an empty curve.
func (c Curve) At(x int64) int64 {
	if c.Len == 0 {
		return 0
	}

	first := c.Points[0]
	if x <= first.X {
		return first.Y
	}

	last := c.Points[c.Len-1]
	if x >= last.X {
		return last.Y
	}

	for i := range c.Len - 1 {
		a, b := c.Points[i], c.Points[i+1]
		if x < a.X || x > b.X {
			continue
		}

		span := b.X - a.X
		if span == 0 {
			return a.Y
		}

		return a.Y + floorDiv((b.Y-a.Y)*(x-a.X), span)
	}

	return last.Y
}
