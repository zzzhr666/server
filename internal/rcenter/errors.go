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
	// ErrCreateBattleRoomFailed means the selected battle node rejected room creation.
	ErrCreateBattleRoomFailed = errors.New("create BattleRoom failed")
	// ErrBattleNodeNotRegistered means rcenter has no cached control client for the node.
	ErrBattleNodeNotRegistered = errors.New("battle node not registered")
	// ErrPlayerInGame means a player is already assigned to an active battle room.
	ErrPlayerInGame = errors.New("player already in game")

	ErrInvalidBattleStats = errors.New("invalid BattleStats")

	ErrInvalidRewardRules = errors.New("invalid reward rules")

	ErrUnavailableCoinClient = errors.New("unavailable coinClient")

	ErrUnavailableGrowthClient = errors.New("unavailable growthClient")

	// ErrActiveMatchNotFound means a player has no resumable battle assignment.
	ErrActiveMatchNotFound = errors.New("active match not found")
)
