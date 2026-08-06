package httpapi

// playerResponse 是 API 返回的玩家 JSON 表示。
type playerResponse struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Coins    int64  `json:"coins"`
}

// errorResponse 是 API 错误响应的 JSON 结构。
type errorResponse struct {
	Error string `json:"error"`
}

type authRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type authLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authSessionResponse struct {
	Token  string         `json:"token"`
	Player playerResponse `json:"player"`
}

type websocketMessage struct {
	Type   string `json:"type"`
	Weapon string `json:"weapon,omitempty"`
}

const (
	messageTypeHeartbeat   = "heartbeat"
	messageTypeMatchStart  = "match_start"
	messageTypeMatchCancel = "match_cancel"
	messageTypeMatchResume = "match_resume"
)

const (
	serverEventMatchResult   = "match_result"
	serverEventMatchError    = "match_error"
	serverEventMatchCanceled = "match_canceled"
)

type friendRequestResponse struct {
	FromPlayerID int64  `json:"from_player_id"`
	ToPlayerID   int64  `json:"to_player_id"`
	CreatedAt    string `json:"created_at"`
}

type friendRequestsResponse struct {
	Requests []friendRequestResponse `json:"requests"`
}

type sendFriendRequestRequest struct {
	ToPlayerID int64 `json:"to_player_id"`
}

type handleFriendRequestRequest struct {
	FromPlayerID int64 `json:"from_player_id"`
}

type deleteFriendRequest struct {
	FriendPlayerID int64 `json:"friend_player_id"`
}

type friendSummaryResponse struct {
	PlayerID  int64  `json:"player_id"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Online    bool   `json:"online"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type friendSummariesResponse struct {
	Friends []friendSummaryResponse `json:"friends"`
}

type friendPresenceChangedMessage struct {
	Type     string `json:"type"`
	PlayerID int64  `json:"player_id"`
	Online   bool   `json:"online"`
	Status   string `json:"status"`
}

type friendRemovedMessage struct {
	Type     string `json:"type"`
	PlayerID int64  `json:"player_id"`
}

type friendRequestReceivedMessage struct {
	Type     string `json:"type"`
	PlayerID int64  `json:"player_id"`
}

type friendRequestHandledMessage struct {
	Type     string `json:"type"`
	PlayerID int64  `json:"player_id"`
}

type connectionReplacedMessage struct {
	Type string `json:"type"`
}

type matchResultMessage struct {
	Type           string `json:"type"`
	Status         string `json:"status"`
	RoomName       string `json:"room_name,omitempty"`
	Token          string `json:"token,omitempty"`
	BattleNodeName string `json:"battle_node_name,omitempty"`
	BattleUDPAddr  string `json:"battle_udp_addr,omitempty"`
}

type matchErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

type matchCancelMessage struct {
	Type string `json:"type"`
}

type growthResponse struct {
	PlayerID         int64                         `json:"player_id"`
	AttackLevel      int32                         `json:"attack_level"`
	AttackSpeedLevel int32                         `json:"attack_speed_level"`
	HealthLevel      int32                         `json:"health_level"`
	MoveSpeedLevel   int32                         `json:"move_speed_level"`
	UpgradeOptions   []growthUpgradeOptionResponse `json:"upgrade_options"`
}

type growthUpgradeOptionResponse struct {
	Type         string `json:"type"`
	CurrentLevel int32  `json:"current_level"`
	NextCost     int64  `json:"next_cost"`
	MaxLevel     int32  `json:"max_level"`
}

type upgradeGrowthRequest struct {
	Type string `json:"type"`
}

type upgradeGrowthResponse struct {
	Growth         growthResponse `json:"growth"`
	RemainingCoins int64          `json:"remaining_coins"`
	Cost           int64          `json:"cost"`
}
