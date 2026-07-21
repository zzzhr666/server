package httpapi

// playerResponse is the JSON representation returned for a player.
type playerResponse struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// errorResponse is the JSON body returned for API errors.
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
	Type string `json:"type"`
}

const (
	messageTypeHeartbeat   = "heartbeat"
	messageTypeMatchStart  = "match_start"
	messageTypeMatchCancel = "match_cancel"
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
	BattleKCPAddr  string `json:"battle_kcp_addr,omitempty"`
}

type matchErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

type matchCancelMessage struct {
	Type string `json:"type"`
}
