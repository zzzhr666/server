package player

import "errors"

var (
	ErrInvalidNickname = errors.New("invalid player nickname")
	ErrNotFound        = errors.New("player not found")
)
