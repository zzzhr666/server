package presence

import "time"

const (
	// StatusOnline is the persisted presence status for connected players.
	StatusOnline = "online"
	// StatusOffline is the status reported when a player disconnects.
	StatusOffline = "offline"
)

// DefaultTTL bounds stale online records when a connection is lost.
const DefaultTTL = 2 * time.Minute

// Presence describes a player's current logic-server connection.
type Presence struct {
	PlayerID   int64
	ServerName string
	Status     string
	UpdatedAt  time.Time
}
