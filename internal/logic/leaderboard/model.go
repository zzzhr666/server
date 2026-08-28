package leaderboard

// Type 标识排行榜的计分维度。
type Type string

const (
	// TypeSoloClearTime 表示单人模式最短纯战斗时间榜。
	TypeSoloClearTime Type = "solo_clear_time"
	// TypeDuoClearTime 表示双人队伍最短纯战斗时间榜。
	TypeDuoClearTime  Type = "duo_clear_time"
	TypeTrioClearTime Type = "trio_clear_time"
	TypeQuadClearTime Type = "quad_clear_time"
	// TypeTotalKills 表示跨模式累计总击杀榜。
	TypeTotalKills Type = "total_kills"
)

// ListInput 描述一次排行榜查询。
type ListInput struct {
	Type       Type
	MapVersion string
	Limit      int64
}

// Player 描述排行榜条目中的玩家展示资料。
type Player struct {
	PlayerID int64
	Nickname string
	Avatar   string
}

// Entry 表示一条排行榜记录。
// Score 在通关时间榜中为毫秒数，在总击杀榜中为累计击杀数。
type Entry struct {
	Rank    int64
	Players []Player
	Score   int64
}

// Result 返回指定类型和地图版本的排行榜记录。
type Result struct {
	Type       Type
	MapVersion string
	Entries    []Entry
}
