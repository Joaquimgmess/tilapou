package farm

import "errors"

// Farm errors: read with no farm, write conflict and invalid action.
var (
	ErrNotFound        = errors.New("farm: not found")
	ErrStaleRevision   = errors.New("farm: state changed underneath the write")
	ErrAlreadyApplied  = errors.New("farm: action already recorded by another writer")
	ErrUnknownReason   = errors.New("farm: reject reason stored in the database is outside the enum")
	ErrUnknownAction   = errors.New("farm: acao desconhecida")
	ErrMissingAuto     = errors.New("farm: buy_upgrade precisa do campo auto (comedouro, aerador, peao, tecnico, contrato)")
	ErrMissingTankKind = errors.New("farm: buy_tank precisa do campo tank_kind")
	ErrMissingTank     = errors.New("farm: essa acao precisa do campo tank_id")
)
