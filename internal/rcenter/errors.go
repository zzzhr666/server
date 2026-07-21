package rcenter

import "errors"

var (
	// ErrInvalidBattleNode means a battle node registration is missing required fields.
	ErrInvalidBattleNode = errors.New("invalid BattleNode")
	// ErrInvalidPlayerID means a player identifier is empty or non-positive.
	ErrInvalidPlayerID = errors.New("invalid PlayerID")
	// ErrNoAvailableBattleNode means no registered battle node can host more players.
	ErrNoAvailableBattleNode = errors.New("no available BattleNode")
	// ErrPlayerNotWaiting means a cancel request targeted a player outside the queue.
	ErrPlayerNotWaiting = errors.New("player not waiting")
)
