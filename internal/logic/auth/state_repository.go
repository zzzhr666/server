package auth

import (
	"context"
	"errors"
	"server/internal/contract/state"
	playerpkg "server/internal/logic/player"
)

type StateRepository struct {
	stateClient state.Client
}

// NewStateRepository 使用 state-server 客户端创建认证仓储。
func NewStateRepository(client state.Client) *StateRepository {
	return &StateRepository{
		stateClient: client,
	}
}

// RegisterAccount 通过 state-server 创建账号、玩家和初始会话。
func (s *StateRepository) RegisterAccount(ctx context.Context, input RegisterAccountInput) (*RegisterAccountResult, error) {
	result, err := s.stateClient.RegisterAccount(ctx, state.RegisterAccountInput{
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

// CreateAccount 通过 state-server 存储账号凭据。
func (s *StateRepository) CreateAccount(ctx context.Context, account *Account) error {
	return mapStateError(s.stateClient.CreateAccount(ctx, toStateAccount(account)))
}

// GetAccount 通过 state-server 读取账号凭据。
func (s *StateRepository) GetAccount(ctx context.Context, username string) (*Account, error) {
	account, err := s.stateClient.GetAccount(ctx, username)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateAccount(account), nil
}

// CreateSession 通过 state-server 存储登录会话。
func (s *StateRepository) CreateSession(ctx context.Context, session *Session) error {
	return mapStateError(s.stateClient.CreateSession(ctx, toStateSession(session)))
}

// GetSession 通过 state-server 读取登录会话。
func (s *StateRepository) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.stateClient.GetSession(ctx, token)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateSession(session), nil
}

// DeleteSession 通过 state-server 删除登录会话。
func (s *StateRepository) DeleteSession(ctx context.Context, token string) error {
	return mapStateError(s.stateClient.DeleteSession(ctx, token))
}

func toStateAccount(account *Account) *state.Account {
	return &state.Account{
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		PlayerID:     account.PlayerID,
	}
}

func fromStateAccount(account *state.Account) *Account {
	if account == nil {
		return nil
	}
	return &Account{
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		PlayerID:     account.PlayerID,
	}
}

func toStateSession(session *Session) *state.Session {
	return &state.Session{
		Token:     session.Token,
		PlayerID:  session.PlayerID,
		ExpiresAt: session.ExpiresAt,
	}
}

func fromStateSession(session *state.Session) *Session {
	if session == nil {
		return nil
	}
	return &Session{
		Token:     session.Token,
		PlayerID:  session.PlayerID,
		ExpiresAt: session.ExpiresAt,
	}
}

func fromStateRegisterAccountResult(result *state.RegisterAccountResult) *RegisterAccountResult {
	if result == nil {
		return nil
	}
	return &RegisterAccountResult{
		Account: fromStateAccount(result.Account),
		Player:  fromStatePlayer(result.Player),
		Session: fromStateSession(result.Session),
	}
}

func fromStatePlayer(player *state.Player) *playerpkg.Player {
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
	case errors.Is(err, state.ErrAccountExists):
		return ErrAccountExists
	case errors.Is(err, state.ErrAccountNotFound):
		return ErrAccountNotFound
	case errors.Is(err, state.ErrSessionNotFound):
		return ErrSessionNotFound
	default:
		return err
	}
}
