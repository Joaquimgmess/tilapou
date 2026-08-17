package balance

import "errors"

// Balance reading errors: a name outside the enum, a key nobody reads or a table longer than the game reads.
var (
	ErrUnknownTankKind   = errors.New("balance: unknown tank kind")
	ErrUnknownAutomation = errors.New("balance: unknown automation")
	ErrUnusedKeys        = errors.New("balance: o arquivo tem chaves que ninguem le")
	ErrTooManyRows       = errors.New("balance: a tabela tem mais linhas do que o jogo le")
)
