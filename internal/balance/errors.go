package balance

import "errors"

// Erros de leitura do balance: nome fora do enum ou chave que ninguem le.
var (
	ErrUnknownTankKind   = errors.New("balance: unknown tank kind")
	ErrUnknownAutomation = errors.New("balance: unknown automation")
	ErrUnusedKeys        = errors.New("balance: o arquivo tem chaves que ninguem le")
)
