package state

import (
	"context"
	"time"
)

type Account struct {
	Username     string
	PasswordHash string
	PlayerID     int64
}

type Session struct {
	Token     string
	PlayerID  int64
	ExpiresAt time.Time
}

type Player struct {
	ID       int64
	Nickname string
	Avatar   string
	Email    string
	Phone    string
}

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

type RegisterAccountResult struct {
	Account *Account
	Player  *Player
	Session *Session
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
