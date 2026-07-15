package player

import "errors"

var (
	ErrInvalidName = errors.New("invalid player name")
	ErrNotFound    = errors.New("player not found")
)
