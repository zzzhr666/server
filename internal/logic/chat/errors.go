package chat

import "errors"

var (
	ErrInvalidPlayerID = errors.New("invalid player id")
	ErrInvalidMessage  = errors.New("invalid chat message")
	ErrInvalidChannel  = errors.New("invalid chat channel")
	ErrMessageNotFound = errors.New("chat message not found")
	ErrMessageExists   = errors.New("chat message exists")
	ErrFriendRequired  = errors.New("friend required")
)
