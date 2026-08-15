package balance

import "errors"

var (
	ErrUnknownTankKind   = errors.New("balance: unknown tank kind")
	ErrUnknownAutomation = errors.New("balance: unknown automation")
	ErrUnusedKeys        = errors.New("balance: o arquivo tem chaves que ninguem le")
)
