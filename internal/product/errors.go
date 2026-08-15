package product

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("product not found")

type InvalidError struct {
	Reason string
}

func (e InvalidError) Error() string {
	return "invalid product: " + e.Reason
}

func invalidf(format string, args ...any) InvalidError {
	return InvalidError{Reason: fmt.Sprintf(format, args...)}
}
