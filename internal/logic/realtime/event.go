package realtime

type Event struct {
	Type           string `json:"type"`
	TargetPlayerID int64  `json:"target_player_id"`
	ActorPlayerID  int64  `json:"actor_player_id"`
	Online         bool   `json:"online,omitempty"`
	Status         string `json:"status,omitempty"`
}

const (
	EventFriendPresenceChanged = "friend_presence_changed"
	EventFriendRemoved         = "friend_removed"
)
