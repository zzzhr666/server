package auth

import "time"

// Account stores login credentials and the player bound to the username.
type Account struct {
	Username     string
	PasswordHash string
	PlayerID     int64
}

// Session represents a login session identified by an opaque token.
type Session struct {
	Token     string
	PlayerID  int64
	ExpiresAt time.Time
}

// RegisterInput contains account credentials and the player profile to create.
type RegisterInput struct {
	Username      string
	PlainPassword string
	Name          string
	Avatar        string
	Email         string
	Phone         string
}

// LoginInput contains account credentials used during login.
type LoginInput struct {
	Username      string
	PlainPassword string
}
