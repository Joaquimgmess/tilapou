package sim

const maxCurvePoints = 12

// CurvePoint e um par de ancoragem de uma Curve.
type CurvePoint struct {
	X int64
	Y int64
}

// Curve e linear por partes, com ate maxCurvePoints pontos ordenados por X.
type Curve struct {
	Points [maxCurvePoints]CurvePoint
	Len    int32
}

// NewCurve erra com ErrCurveEmpty sem pontos, ErrCurveTooLong acima do limite e ErrCurveNotSorted se X nao crescer.
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

// At interpola Y em x, prendendo nas extremidades; devolve 0 na curva vazia.
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
