package rcenter

import "time"

// BattleNode 描述一个可承载已匹配玩家的战斗进程。
type BattleNode struct {
	Name          string
	UDPAddr       string
	ControlAddr   string
	MaxPlayers    int
	ActivePlayers int
	LastSeen      time.Time
}

// MatchStatus 是返回给 logic-server 的匹配状态。
type MatchStatus string

const (
	// MatchStatusWaiting 表示玩家正在队列中等待对手。
	MatchStatusWaiting MatchStatus = "waiting"
	// MatchStatusMatched 表示已分配房间和战斗节点。
	MatchStatusMatched MatchStatus = "matched"
	// MatchStatusUnexpected 保留远程响应中的未知状态值。
	MatchStatusUnexpected MatchStatus = "unexpected"
)

// MatchResult 包含发送给已匹配玩家的匹配结果。
type MatchResult struct {
	Status         MatchStatus
	RoomName       string
	Token          string
	BattleNodeName string
	BattleUDPAddr  string
	PlayerIDs      []int64
	PlayerLoadouts []PlayerLoadout
}

// CreateBattleRoomInput 包含发送给战斗节点的房间预留数据。
type CreateBattleRoomInput struct {
	RoomName       string
	Token          string
	PlayerIDs      []int64
	PlayerLoadouts []PlayerLoadout
}

type PlayerLoadout struct {
	PlayerID         int64
	Weapon           string
	AttackLevel      int32
	AttackSpeedLevel int32
	HealthLevel      int32
	MoveSpeedLevel   int32
}

// FinishMatchInput 描述战斗服上报的一次对局结束、权威战斗统计和纯战斗耗时。
type FinishMatchInput struct {
	RoomName         string
	PlayerIDs        []int64
	Reason           string
	PlayerStats      []PlayerBattleStats
	CombatDurationMS int64
}

// ActiveMatch 存储已匹配玩家可以恢复的战斗连接数据。
type ActiveMatch struct {
	RoomName       string
	Token          string
	BattleNodeName string
	BattleUDPAddr  string
	PlayerIDs      []int64
	PlayerLoadouts []PlayerLoadout
	CreatedAt      time.Time
}
