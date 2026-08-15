package sim

import "errors"

var (
	ErrCurveEmpty     = errors.New("sim: curve needs at least one point")
	ErrCurveTooLong   = errors.New("sim: curve exceeds the maximum number of points")
	ErrCurveNotSorted = errors.New("sim: curve points must be strictly increasing in x")

	ErrBalanceUnversioned         = errors.New("sim: balance must carry a version")
	ErrBalanceNoTempCurve         = errors.New("sim: balance needs a temperature multiplier curve")
	ErrBalanceNoMass              = errors.New("sim: balance needs positive harvest and maximum mass")
	ErrBalanceNoRation            = errors.New("sim: balance needs at least one ration step")
	ErrBalanceNoMarket            = errors.New("sim: balance needs market base prices and a period")
	ErrBalanceTankSlotEmpty       = errors.New("sim: every tank kind needs a spec")
	ErrBalanceAutomationSlotEmpty = errors.New("sim: every automation needs a cost")

	ErrNoBalance       = errors.New("sim: advance needs a balance")
	ErrBackwardsTime   = errors.New("sim: cannot advance to a tick in the past")
	ErrActionsUnsorted = errors.New("sim: actions must be sorted by tick")
)
