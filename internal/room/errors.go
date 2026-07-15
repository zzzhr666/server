package room

import "errors"

var (
	ErrNotFound        = errors.New("room not found")
	ErrPlayerNotFound  = errors.New("player not found")
	ErrPlayerAlreadyIn = errors.New("player already in room")
	ErrPlayerNotIn     = errors.New("player not in room")
)
