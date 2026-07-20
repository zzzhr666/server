package state

import (
	"context"
	"time"
)

// Account stores login credentials and the bound player ID.
type Account struct {
	Username     string
	PasswordHash string
	PlayerID     int64
}

// Session stores an authenticated player session.
type Session struct {
	Token     string
	PlayerID  int64
	ExpiresAt time.Time
}

// Player stores user-visible player profile data.
type Player struct {
	ID       int64
	Nickname string
	Avatar   string
	Email    string
	Phone    string
}

// RegisterAccountInput groups the state data needed for account registration.
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

// RegisterAccountResult returns all records created during registration.
type RegisterAccountResult struct {
	Account *Account
	Player  *Player
	Session *Session
}

// Presence records where a player is currently connected.
type Presence struct {
	PlayerID   int64
	ServerName string
	Status     string
	UpdatedAt  time.Time
}

// Client defines state-server operations needed by other processes.
type Client interface {
	CreateAccount(ctx context.Context, account *Account) error
	GetAccount(ctx context.Context, username string) (*Account, error)
	RegisterAccount(ctx context.Context, input RegisterAccountInput) (*RegisterAccountResult, error)

	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error

	CreatePlayer(ctx context.Context, player *Player) error
	GetPlayer(ctx context.Context, id int64) (*Player, error)
	NextPlayerID(ctx context.Context) (int64, error)
}

// PresenceClient defines state-server operations for player online state.
type PresenceClient interface {
	SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error
}
