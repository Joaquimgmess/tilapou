package balance

import "errors"

// Balance reading errors: a name outside the enum or a key nobody reads.
var (
	ErrUnknownTankKind   = errors.New("balance: unknown tank kind")
	ErrUnknownAutomation = errors.New("balance: unknown automation")
	ErrUnusedKeys        = errors.New("balance: o arquivo tem chaves que ninguem le")
)
