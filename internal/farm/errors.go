package farm

import "errors"

var (
	ErrNotFound        = errors.New("farm: not found")
	ErrStaleRevision   = errors.New("farm: state changed underneath the write")
	ErrUnknownAction   = errors.New("farm: acao desconhecida")
	ErrMissingAuto     = errors.New("farm: buy_upgrade precisa do campo auto (comedouro, aerador, peao, tecnico, contrato)")
	ErrMissingTankKind = errors.New("farm: buy_tank precisa do campo tank_kind")
	ErrMissingTank     = errors.New("farm: essa acao precisa do campo tank_id")
	ErrBadAmount       = errors.New("farm: amount must be positive")
)

func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
