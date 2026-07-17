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
