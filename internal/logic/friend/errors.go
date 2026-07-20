package friend

import "errors"

var (
	ErrInvalidPlayerID = errors.New("invalid player id")
	ErrInvalidRequest  = errors.New("invalid friend request")
	ErrRequestExists   = errors.New("friend request exists")
	ErrRequestNotFound = errors.New("friend request not found")
	ErrAlreadyExists   = errors.New("friend already exists")
	ErrNotFound        = errors.New("friend not found")
)
