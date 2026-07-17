package auth

import (
	"server/internal/logic/player"
	"time"
)

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

type AuthorizeResult struct {
	Session *Session
	Player  *player.Player
}

// RegisterInput contains account credentials and the player profile to create.
type RegisterInput struct {
	Username      string
	PlainPassword string
	Nickname      string
	Avatar        string
	Email         string
	Phone         string
}

// RegisterAccountInput contains validated account, player, and session data for atomic registration.
type RegisterAccountInput struct {
	Username         string
	PasswordHash     string
	Nickname         string
	Avatar           string
	Email            string
	Phone            string
	SessionToken     string
	SessionExpiresAt time.Time
}

// RegisterAccountResult contains the account, player, and session created during registration.
type RegisterAccountResult struct {
	Account *Account
	Player  *player.Player
	Session *Session
}

// LoginInput contains account credentials used during login.
type LoginInput struct {
	Username      string
	PlainPassword string
}
