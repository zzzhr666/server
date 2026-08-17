package rcenter

import "errors"

var (
	// ErrInvalidBattleNode 表示战斗节点注册缺少必填字段。
	ErrInvalidBattleNode = errors.New("invalid BattleNode")
	// ErrInvalidPlayerID 表示玩家标识为空或非正数。
	ErrInvalidPlayerID = errors.New("invalid PlayerID")
	// ErrInvalidRoomName 表示对局结束请求缺少稳定的房间标识。
	ErrInvalidRoomName = errors.New("invalid room name")
	// ErrNoAvailableBattleNode 表示没有已注册战斗节点可以承载更多玩家。
	ErrNoAvailableBattleNode = errors.New("no available BattleNode")
	// ErrPlayerNotWaiting 表示取消请求指向了队列外的玩家。
	ErrPlayerNotWaiting = errors.New("player not waiting")
	// ErrCreateBattleRoomFailed 表示选定战斗节点拒绝创建房间。
	ErrCreateBattleRoomFailed = errors.New("create BattleRoom failed")
	// ErrBattleNodeNotRegistered 表示 rcenter 没有该节点的缓存控制客户端。
	ErrBattleNodeNotRegistered = errors.New("battle node not registered")
	// ErrPlayerInGame 表示玩家已被分配到活跃战斗房间。
	ErrPlayerInGame = errors.New("player already in game")

	ErrInvalidBattleStats = errors.New("invalid BattleStats")

	ErrInvalidRewardRules = errors.New("invalid reward rules")

	ErrUnavailableCoinClient = errors.New("unavailable coinClient")

	ErrUnavailableGrowthClient = errors.New("unavailable growthClient")

	// ErrActiveMatchNotFound 表示玩家没有可恢复的战斗分配。
	ErrActiveMatchNotFound = errors.New("active match not found")
)
