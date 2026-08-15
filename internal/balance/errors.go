package balance

import "errors"

var (
	ErrUnknownTankKind   = errors.New("balance: unknown tank kind")
	ErrUnknownAutomation = errors.New("balance: unknown automation")
)
