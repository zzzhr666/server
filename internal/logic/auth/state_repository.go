package auth

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	playerpkg "server/internal/logic/player"
)

type StateRepository struct {
	stateClient statecontract.Client
}

// NewStateRepository creates an auth repository backed by a state-server client.
func NewStateRepository(client statecontract.Client) *StateRepository {
	return &StateRepository{
		stateClient: client,
	}
}

// RegisterAccount creates account, player, and initial session through state-server.
func (s *StateRepository) RegisterAccount(ctx context.Context, input RegisterAccountInput) (*RegisterAccountResult, error) {
	result, err := s.stateClient.RegisterAccount(ctx, statecontract.RegisterAccountInput{
		Username:         input.Username,
		PasswordHash:     input.PasswordHash,
		Nickname:         input.Nickname,
		Avatar:           input.Avatar,
		Email:            input.Email,
		Phone:            input.Phone,
		SessionToken:     input.SessionToken,
		SessionExpiresAt: input.SessionExpiresAt,
	})
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateRegisterAccountResult(result), nil
}

// CreateAccount stores account credentials through state-server.
func (s *StateRepository) CreateAccount(ctx context.Context, account *Account) error {
	return mapStateError(s.stateClient.CreateAccount(ctx, toStateAccount(account)))
}

// GetAccount loads account credentials through state-server.
func (s *StateRepository) GetAccount(ctx context.Context, username string) (*Account, error) {
	account, err := s.stateClient.GetAccount(ctx, username)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateAccount(account), nil
}

// CreateSession stores a login session through state-server.
func (s *StateRepository) CreateSession(ctx context.Context, session *Session) error {
	return mapStateError(s.stateClient.CreateSession(ctx, toStateSession(session)))
}

// GetSession loads a login session through state-server.
func (s *StateRepository) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.stateClient.GetSession(ctx, token)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateSession(session), nil
}

// DeleteSession removes a login session through state-server.
func (s *StateRepository) DeleteSession(ctx context.Context, token string) error {
	return mapStateError(s.stateClient.DeleteSession(ctx, token))
}

func toStateAccount(account *Account) *statecontract.Account {
	return &statecontract.Account{
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		PlayerID:     account.PlayerID,
	}
}

func fromStateAccount(account *statecontract.Account) *Account {
	if account == nil {
		return nil
	}
	return &Account{
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		PlayerID:     account.PlayerID,
	}
}

func toStateSession(session *Session) *statecontract.Session {
	return &statecontract.Session{
		Token:     session.Token,
		PlayerID:  session.PlayerID,
		ExpiresAt: session.ExpiresAt,
	}
}

func fromStateSession(session *statecontract.Session) *Session {
	if session == nil {
		return nil
	}
	return &Session{
		Token:     session.Token,
		PlayerID:  session.PlayerID,
		ExpiresAt: session.ExpiresAt,
	}
}

func fromStateRegisterAccountResult(result *statecontract.RegisterAccountResult) *RegisterAccountResult {
	if result == nil {
		return nil
	}
	return &RegisterAccountResult{
		Account: fromStateAccount(result.Account),
		Player:  fromStatePlayer(result.Player),
		Session: fromStateSession(result.Session),
	}
}

func fromStatePlayer(player *statecontract.Player) *playerpkg.Player {
	if player == nil {
		return nil
	}
	return &playerpkg.Player{
		ID:       player.ID,
		Nickname: player.Nickname,
		Avatar:   player.Avatar,
		Email:    player.Email,
		Phone:    player.Phone,
	}
}

func mapStateError(err error) error {
	switch {
	case errors.Is(err, statecontract.ErrAccountExists):
		return ErrAccountExists
	case errors.Is(err, statecontract.ErrAccountNotFound):
		return ErrAccountNotFound
	case errors.Is(err, statecontract.ErrSessionNotFound):
		return ErrSessionNotFound
	default:
		return err
	}
}
