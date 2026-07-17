package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"server/internal/logic/player"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service defines account authentication and session operations.
type Service interface {
	// Register creates an account, creates the bound player, and returns a session.
	Register(ctx context.Context, input RegisterInput) (*AuthorizeResult, error)
	// Login validates credentials and creates a new session.
	Login(ctx context.Context, input LoginInput) (*AuthorizeResult, error)
	// Logout deletes a login session by token.
	Logout(ctx context.Context, token string) error
	// GetSession returns a valid login session by token.
	GetSession(ctx context.Context, token string) (*Session, error)
	// GetCurrentPlayer returns the player bound to a valid session token.
	GetCurrentPlayer(ctx context.Context, token string) (*player.Player, error)
}

// Repository defines auth persistence operations used by the service layer.
type Repository interface {
	// RegisterAccount creates account, player, and first session in one state operation.
	RegisterAccount(ctx context.Context, input RegisterAccountInput) (*RegisterAccountResult, error)
	// GetAccount loads an account by username.
	GetAccount(ctx context.Context, username string) (*Account, error)
	// CreateSession stores a login session with its expiration.
	CreateSession(ctx context.Context, session *Session) error
	// GetSession loads a non-expired session by token.
	GetSession(ctx context.Context, token string) (*Session, error)
	// DeleteSession removes a session token.
	DeleteSession(ctx context.Context, token string) error
}

// NewService creates an auth service with auth storage, player creation, and session TTL.
func NewService(authRepo Repository, playerService player.Service, sessionTTL time.Duration) *GameAuthService {
	return &GameAuthService{
		authRepo:      authRepo,
		playerService: playerService,
		sessionTTL:    sessionTTL,
	}
}

// GameAuthService implements account registration, login, and session rules.
type GameAuthService struct {
	authRepo      Repository
	playerService player.Service
	sessionTTL    time.Duration
}

// Register creates a player-backed account and immediately creates a login session.
func (g *GameAuthService) Register(ctx context.Context, input RegisterInput) (*AuthorizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Username == "" {
		return nil, ErrInvalidUsername
	}
	if input.PlainPassword == "" {
		return nil, ErrInvalidPassword
	}
	if input.Nickname == "" {
		return nil, player.ErrInvalidNickname
	}

	passwordHash, err := hashPassword(input.PlainPassword)
	if err != nil {
		return nil, err
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	result, err := g.authRepo.RegisterAccount(ctx, RegisterAccountInput{
		Username:         input.Username,
		PasswordHash:     passwordHash,
		Nickname:         input.Nickname,
		Avatar:           input.Avatar,
		Email:            input.Email,
		Phone:            input.Phone,
		SessionToken:     token,
		SessionExpiresAt: time.Now().Add(g.sessionTTL),
	})
	if err != nil {
		return nil, err
	}
	return &AuthorizeResult{
		Session: result.Session,
		Player:  result.Player,
	}, nil
}

// Login validates username and password then creates a fresh login session.
func (g *GameAuthService) Login(ctx context.Context, input LoginInput) (*AuthorizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Username == "" {
		return nil, ErrInvalidUsername
	}
	if input.PlainPassword == "" {
		return nil, ErrInvalidPassword
	}
	account, err := g.authRepo.GetAccount(ctx, input.Username)
	if errors.Is(err, ErrAccountNotFound) {
		return nil, ErrInvalidCredentials
	} else if err != nil {
		return nil, err
	}
	if correct := checkPassword(account.PasswordHash, input.PlainPassword); !correct {
		return nil, ErrInvalidCredentials
	}

	p, err := g.playerService.Get(ctx, account.PlayerID)
	if err != nil {
		return nil, err
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	session := &Session{
		Token:     token,
		PlayerID:  account.PlayerID,
		ExpiresAt: time.Now().Add(g.sessionTTL),
	}

	if err := g.authRepo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &AuthorizeResult{
		Session: session,
		Player:  p,
	}, nil
}

// Logout deletes the session identified by token.
func (g *GameAuthService) Logout(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return ErrSessionNotFound
	}
	return g.authRepo.DeleteSession(ctx, token)
}

// GetSession returns a stored session for a non-empty token.
func (g *GameAuthService) GetSession(ctx context.Context, token string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if token == "" {
		return nil, ErrSessionNotFound
	}
	return g.authRepo.GetSession(ctx, token)
}

// GetCurrentPlayer returns the player bound to a valid session token.
func (g *GameAuthService) GetCurrentPlayer(ctx context.Context, token string) (*player.Player, error) {
	session, err := g.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return g.playerService.Get(ctx, session.PlayerID)
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func checkPassword(passwordHash, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plainPassword))
	return err == nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
