package presence

import "time"

const (
	// StatusOnline 是已连接玩家持久化的在线状态。
	StatusOnline = "online"
	// StatusOffline 是玩家断开连接时上报的状态。
	StatusOffline = "offline"
)

// DefaultTTL 用于限制连接丢失后残留在线记录的最长时间。
const DefaultTTL = 2 * time.Minute

// Presence 描述玩家当前连接的 logic-server。
type Presence struct {
	PlayerID   int64
	ServerName string
	Status     string
	UpdatedAt  time.Time
}
