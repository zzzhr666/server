package state

import "errors"

var (
	ErrAccountExists        = errors.New("state account exists")
	ErrAccountNotFound      = errors.New("state account not found")
	ErrSessionNotFound      = errors.New("state session not found")
	ErrPlayerNotFound       = errors.New("state player not found")
	ErrPresenceNotFound     = errors.New("state presence not found")
	ErrInvalidPresence      = errors.New("invalid state presence")
	ErrInvalidRealtimeRoute = errors.New("invalid realtime route")

	ErrFriendRequestExists   = errors.New("friend request exists")
	ErrFriendRequestNotFound = errors.New("friend request does not exist")
	ErrFriendAlreadyExists   = errors.New("friend already exists")
	ErrFriendNotFound        = errors.New("friend not found")
	ErrInvalidFriendRequest  = errors.New("invalid friend request")

	ErrGrowthNotFound     = errors.New("growth not found")
	ErrInvalidGrowth      = errors.New("invalid growth")
	ErrInvalidGrowthField = errors.New("invalid growth field")
	ErrInsufficientCoins  = errors.New("insufficient coins")
	ErrMaxGrowthLevel     = errors.New("max growth level")
	ErrInvalidSettlement  = errors.New("invalid settlement")

	ErrInvalidPlayer = errors.New("invalid player")

	ErrChatMessageNotFound = errors.New("chat message not found")
	ErrInvalidChatMessage  = errors.New("invalid chat message")
	ErrInvalidChatChannel  = errors.New("invalid chat channel")
	ErrChatMessageExists   = errors.New("chat message exists")
)
