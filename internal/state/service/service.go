package service

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"sync"
)

type accountStore interface {
	CreateAccount(ctx context.Context, account *statecontract.Account) error
	GetAccount(ctx context.Context, username string) (*statecontract.Account, error)
}

type sessionStore interface {
	CreateSession(ctx context.Context, account *statecontract.Session) error
	GetSession(ctx context.Context, token string) (*statecontract.Session, error)
	DeleteSession(ctx context.Context, token string) error
}

type playerStore interface {
	CreatePlayer(ctx context.Context, player *statecontract.Player) error
	GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error)
	NextPlayerID(ctx context.Context) (int64, error)
}

type Service struct {
	accounts accountStore
	sessions sessionStore
	players  playerStore

	accountMu sync.RWMutex
	sessionMu sync.RWMutex
	playerMu  sync.RWMutex
}

func (s *Service) CreatePlayer(ctx context.Context, player *statecontract.Player) error {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	return s.players.CreatePlayer(ctx, player)
}

func (s *Service) GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error) {
	s.playerMu.RLock()
	defer s.playerMu.RUnlock()
	return s.players.GetPlayer(ctx, id)
}

func (s *Service) NextPlayerID(ctx context.Context) (int64, error) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	return s.players.NextPlayerID(ctx)
}

func (s *Service) CreateSession(ctx context.Context, session *statecontract.Session) error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	return s.sessions.CreateSession(ctx, session)
}

func (s *Service) GetSession(ctx context.Context, token string) (*statecontract.Session, error) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.sessions.GetSession(ctx, token)
}

func (s *Service) DeleteSession(ctx context.Context, token string) error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	return s.sessions.DeleteSession(ctx, token)
}

func (s *Service) CreateAccount(ctx context.Context, account *statecontract.Account) error {
	s.accountMu.Lock()
	defer s.accountMu.Unlock()
	return s.accounts.CreateAccount(ctx, account)
}

func (s *Service) GetAccount(ctx context.Context, username string) (*statecontract.Account, error) {
	s.accountMu.RLock()
	defer s.accountMu.RUnlock()
	return s.accounts.GetAccount(ctx, username)
}

func (s *Service) RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	s.accountMu.Lock()
	defer s.accountMu.Unlock()

	s.playerMu.Lock()
	defer s.playerMu.Unlock()

	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	account, err := s.accounts.GetAccount(ctx, input.Username)
	if err == nil && account != nil {
		return nil, statecontract.ErrAccountExists
	} else if err != nil && !errors.Is(err, statecontract.ErrAccountNotFound) {
		return nil, err
	}

	playerID, err := s.players.NextPlayerID(ctx)
	if err != nil {
		return nil, err
	}
	player := &statecontract.Player{
		ID:       playerID,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}

	if err := s.players.CreatePlayer(ctx, player); err != nil {
		return nil, err
	}

	account = &statecontract.Account{
		Username:     input.Username,
		PasswordHash: input.PasswordHash,
		PlayerID:     playerID,
	}
	if err := s.accounts.CreateAccount(ctx, account); err != nil {
		return nil, err
	}
	session := &statecontract.Session{
		Token:     input.SessionToken,
		PlayerID:  playerID,
		ExpiresAt: input.SessionExpiresAt,
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &statecontract.RegisterAccountResult{
		Account: account,
		Player:  player,
		Session: session,
	}, nil
}

func NewService(accounts accountStore, sessions sessionStore, players playerStore) *Service {
	return &Service{
		accounts: accounts,
		sessions: sessions,
		players:  players,
	}
}
