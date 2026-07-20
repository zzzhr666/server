package state

import "errors"

var (
	ErrAccountExists    = errors.New("state account exists")
	ErrAccountNotFound  = errors.New("state account not found")
	ErrSessionNotFound  = errors.New("state session not found")
	ErrPlayerNotFound   = errors.New("state player not found")
	ErrPresenceNotFound = errors.New("state presence not found")
	ErrInvalidPresence  = errors.New("invalid state presence")

	ErrFriendRequestExists   = errors.New("friend request exists")
	ErrFriendRequestNotFound = errors.New("friend request does not exist")
	ErrFriendAlreadyExists   = errors.New("friend already exists")
	ErrFriendNotFound        = errors.New("friend not found")
	ErrInvalidFriendRequest  = errors.New("invalid friend request")
)
