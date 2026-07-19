package state

import "errors"

var (
	ErrAccountExists    = errors.New("state account exists")
	ErrAccountNotFound  = errors.New("state account not found")
	ErrSessionNotFound  = errors.New("state session not found")
	ErrPlayerNotFound   = errors.New("state player not found")
	ErrPresenceNotFound = errors.New("state presence not found")
	ErrInvalidPresence  = errors.New("invalid state presence")
)
