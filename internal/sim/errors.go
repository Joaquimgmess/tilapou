package sim

import "errors"

var (
	ErrCurveEmpty     = errors.New("sim: curve needs at least one point")
	ErrCurveTooLong   = errors.New("sim: curve exceeds the maximum number of points")
	ErrCurveNotSorted = errors.New("sim: curve points must be strictly increasing in x")
)
