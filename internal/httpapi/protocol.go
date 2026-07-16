package httpapi

// createPlayerRequest is the JSON body for creating a player.
type createPlayerRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

// roomPlayerRequest is the JSON body for room actions performed by a player.
type roomPlayerRequest struct {
	PlayerID int64 `json:"player_id"`
}

// playerResponse is the JSON representation returned for a player.
type playerResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

// errorResponse is the JSON body returned for API errors.
type errorResponse struct {
	Error string `json:"error"`
}

// createRoomRequest is the JSON body for creating a room.
type createRoomRequest struct {
	OwnerID    int64 `json:"owner_id"`
	MaxPlayers int   `json:"max_players"`
}

// roomResponse is the JSON representation returned for a room.
type roomResponse struct {
	ID           int64   `json:"id"`
	OwnerID      int64   `json:"owner_id"`
	Status       string  `json:"status"`
	MaxPlayers   int     `json:"max_players"`
	Players      []int64 `json:"players"`
	ReadyPlayers []int64 `json:"ready_players"`
}

// listRoomsResponse is the JSON body returned when listing rooms.
type listRoomsResponse struct {
	Rooms []roomResponse `json:"rooms"`
}

// updatePlayerRequest is the JSON body for partial player profile updates.
type updatePlayerRequest struct {
	Name   *string `json:"name"`
	Avatar *string `json:"avatar"`
	Email  *string `json:"email"`
	Phone  *string `json:"phone"`
}

// roomDetailResponse is the JSON representation of a room with player profiles.
type roomDetailResponse struct {
	ID         int64                `json:"id"`
	OwnerID    int64                `json:"owner_id"`
	Status     string               `json:"status"`
	MaxPlayers int                  `json:"max_players"`
	Players    []roomPlayerResponse `json:"players"`
}

// roomPlayerResponse is the JSON representation of a player inside a room.
type roomPlayerResponse struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	Ready   bool   `json:"ready"`
	IsOwner bool   `json:"owner"`
}
