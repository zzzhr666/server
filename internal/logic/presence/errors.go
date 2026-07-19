package presence

import "errors"

var (
	ErrInvalidPresence = errors.New("invalid presence")
	ErrNotFound        = errors.New("presence not found")
)
