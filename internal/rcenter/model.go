package rcenter

import "time"

// BattleNode describes one battle process that can host matched players.
type BattleNode struct {
	Name          string
	KCPAddr       string
	ControlAddr   string
	MaxPlayers    int
	ActivePlayers int
	LastSeen      time.Time
}

// MatchStatus is the matchmaking state returned to logic servers.
type MatchStatus string

const (
	// MatchStatusWaiting means the player is queued and waiting for an opponent.
	MatchStatusWaiting MatchStatus = "waiting"
	// MatchStatusMatched means a room and battle node were assigned.
	MatchStatusMatched MatchStatus = "matched"
	// MatchStatusUnexpected preserves unknown status values from remote responses.
	MatchStatusUnexpected MatchStatus = "unexpected"
)

// MatchResult contains the matchmaking outcome sent back to matched players.
type MatchResult struct {
	Status         MatchStatus
	RoomName       string
	Token          string
	BattleNodeName string
	BattleKCPAddr  string
	PlayerIDs      []int64
}

// CreateBattleRoomInput contains the room reservation data sent to a battle node.
type CreateBattleRoomInput struct {
	RoomName  string
	Token     string
	PlayerIDs []int64
}
