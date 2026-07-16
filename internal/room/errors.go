package room

import "errors"

var (
	ErrNotFound                   = errors.New("room not found")
	ErrPlayerNotFound             = errors.New("player not found")
	ErrPlayerAlreadyInThisRoom    = errors.New("player already in this room")
	ErrPlayerNotIn                = errors.New("player not in room")
	ErrRoomFull                   = errors.New("room is full")
	ErrRoomNotWaiting             = errors.New("room is not waiting")
	ErrOnlyOwnerCanStart          = errors.New("only owner can start")
	ErrPlayersNotReady            = errors.New("players not ready")
	ErrInvalidMaxPlayers          = errors.New("invalid max players")
	ErrOwnerCannotReadyOrUnready  = errors.New("owner cannot ready or unready")
	ErrPlayerAlreadyInAnotherRoom = errors.New("player already in another room")
)
