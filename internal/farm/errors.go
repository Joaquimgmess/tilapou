package farm

import "errors"

var (
	ErrNotFound      = errors.New("farm: not found")
	ErrStaleRevision = errors.New("farm: state changed underneath the write")
	ErrUnknownAction = errors.New("farm: unknown action")
	ErrBadAmount     = errors.New("farm: amount must be positive")
)

func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
