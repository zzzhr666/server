package auth

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestStateRepositoryAccount(t *testing.T) {
	client := newFakeStateClient()
	repo := NewStateRepository(client)
	account := &Account{
		Username:     "alice",
		PasswordHash: "hash",
		PlayerID:     7,
	}

	if err := repo.CreateAccount(context.Background(), account); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	got, err := repo.GetAccount(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if got.Username != account.Username {
		t.Fatalf("username = %q, want %q", got.Username, account.Username)
	}
	if got.PasswordHash != account.PasswordHash {
		t.Fatalf("password hash = %q, want %q", got.PasswordHash, account.PasswordHash)
	}
	if got.PlayerID != account.PlayerID {
		t.Fatalf("player id = %d, want %d", got.PlayerID, account.PlayerID)
	}
}

func TestStateRepositorySession(t *testing.T) {
	client := newFakeStateClient()
	repo := NewStateRepository(client)
	session := &Session{
		Token:     "token-1",
		PlayerID:  7,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	got, err := repo.GetSession(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if got.Token != session.Token {
		t.Fatalf("token = %q, want %q", got.Token, session.Token)
	}
	if got.PlayerID != session.PlayerID {
		t.Fatalf("player id = %d, want %d", got.PlayerID, session.PlayerID)
	}
	if !got.ExpiresAt.Equal(session.ExpiresAt) {
		t.Fatalf("expires at = %v, want %v", got.ExpiresAt, session.ExpiresAt)
	}

	if err := repo.DeleteSession(context.Background(), "token-1"); err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
	_, err = repo.GetSession(context.Background(), "token-1")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSession error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestStateRepositoryRegisterAccount(t *testing.T) {
	client := newFakeStateClient()
	repo := NewStateRepository(client)
	expiresAt := time.Now().Add(time.Hour)

	result, err := repo.RegisterAccount(context.Background(), RegisterAccountInput{
		Username:         "alice",
		PasswordHash:     "hash",
		Nickname:         "Alice",
		Avatar:           "alice.png",
		Email:            "alice@example.com",
		Phone:            "13800000000",
		SessionToken:     "token-1",
		SessionExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("RegisterAccount returned error: %v", err)
	}
	if result.Account.Username != "alice" {
		t.Fatalf("account username = %q, want alice", result.Account.Username)
	}
	if result.Account.PasswordHash != "hash" {
		t.Fatalf("account password hash = %q, want hash", result.Account.PasswordHash)
	}
	if result.Player.ID != 1 {
		t.Fatalf("player id = %d, want 1", result.Player.ID)
	}
	if result.Player.Nickname != "Alice" || result.Player.Avatar != "alice.png" || result.Player.Email != "alice@example.com" || result.Player.Phone != "13800000000" {
		t.Fatalf("player profile = %#v, want register input profile", result.Player)
	}
	if result.Session.Token != "token-1" {
		t.Fatalf("session token = %q, want token-1", result.Session.Token)
	}
	if result.Session.PlayerID != result.Player.ID {
		t.Fatalf("session player id = %d, want %d", result.Session.PlayerID, result.Player.ID)
	}
	if !result.Session.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("session expires at = %v, want %v", result.Session.ExpiresAt, expiresAt)
	}
}

func TestStateRepositoryPropagatesGetAccountError(t *testing.T) {
	repo := NewStateRepository(newFakeStateClient())

	_, err := repo.GetAccount(context.Background(), "missing")
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("GetAccount error = %v, want %v", err, ErrAccountNotFound)
	}
}

func TestStateRepositoryMapsCreateAccountExists(t *testing.T) {
	client := newFakeStateClient()
	repo := NewStateRepository(client)
	account := &Account{
		Username:     "alice",
		PasswordHash: "hash",
		PlayerID:     7,
	}
	if err := repo.CreateAccount(context.Background(), account); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}

	err := repo.CreateAccount(context.Background(), account)
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("CreateAccount error = %v, want %v", err, ErrAccountExists)
	}
}

func TestStateRepositoryMapsRegisterAccountExists(t *testing.T) {
	client := newFakeStateClient()
	repo := NewStateRepository(client)
	input := RegisterAccountInput{
		Username:         "alice",
		PasswordHash:     "hash",
		Nickname:         "Alice",
		SessionToken:     "token-1",
		SessionExpiresAt: time.Now().Add(time.Hour),
	}
	if _, err := repo.RegisterAccount(context.Background(), input); err != nil {
		t.Fatalf("RegisterAccount returned error: %v", err)
	}

	_, err := repo.RegisterAccount(context.Background(), input)
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("RegisterAccount error = %v, want %v", err, ErrAccountExists)
	}
}

func TestStateRepositoryMapsGetSessionMissing(t *testing.T) {
	repo := NewStateRepository(newFakeStateClient())

	_, err := repo.GetSession(context.Background(), "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSession error = %v, want %v", err, ErrSessionNotFound)
	}
}

type fakeStateClient struct {
	nextPlayerID int64
	accounts     map[string]*statecontract.Account
	sessions     map[string]*statecontract.Session
	players      map[int64]*statecontract.Player
}

func newFakeStateClient() *fakeStateClient {
	return &fakeStateClient{
		nextPlayerID: 1,
		accounts:     make(map[string]*statecontract.Account),
		sessions:     make(map[string]*statecontract.Session),
		players:      make(map[int64]*statecontract.Player),
	}
}

func (f *fakeStateClient) CreateAccount(_ context.Context, account *statecontract.Account) error {
	if _, ok := f.accounts[account.Username]; ok {
		return statecontract.ErrAccountExists
	}
	cp := *account
	f.accounts[account.Username] = &cp
	return nil
}

func (f *fakeStateClient) RegisterAccount(_ context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	if _, ok := f.accounts[input.Username]; ok {
		return nil, statecontract.ErrAccountExists
	}
	player := &statecontract.Player{
		ID:       f.nextPlayerID,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}
	f.nextPlayerID++
	account := &statecontract.Account{
		Username:     input.Username,
		PasswordHash: input.PasswordHash,
		PlayerID:     player.ID,
	}
	session := &statecontract.Session{
		Token:     input.SessionToken,
		PlayerID:  player.ID,
		ExpiresAt: input.SessionExpiresAt,
	}

	accountCopy := *account
	playerCopy := *player
	sessionCopy := *session
	f.accounts[account.Username] = &accountCopy
	f.players[player.ID] = &playerCopy
	f.sessions[session.Token] = &sessionCopy

	return &statecontract.RegisterAccountResult{
		Account: account,
		Player:  player,
		Session: session,
	}, nil
}

func (f *fakeStateClient) GetAccount(_ context.Context, username string) (*statecontract.Account, error) {
	account, ok := f.accounts[username]
	if !ok {
		return nil, statecontract.ErrAccountNotFound
	}
	cp := *account
	return &cp, nil
}

func (f *fakeStateClient) CreateSession(_ context.Context, session *statecontract.Session) error {
	cp := *session
	f.sessions[session.Token] = &cp
	return nil
}

func (f *fakeStateClient) GetSession(_ context.Context, token string) (*statecontract.Session, error) {
	session, ok := f.sessions[token]
	if !ok {
		return nil, statecontract.ErrSessionNotFound
	}
	cp := *session
	return &cp, nil
}

func (f *fakeStateClient) DeleteSession(_ context.Context, token string) error {
	delete(f.sessions, token)
	return nil
}

func (f *fakeStateClient) CreatePlayer(_ context.Context, _ *statecontract.Player) error {
	return nil
}

func (f *fakeStateClient) GetPlayer(_ context.Context, _ int64) (*statecontract.Player, error) {
	return nil, nil
}

func (f *fakeStateClient) NextPlayerID(_ context.Context) (int64, error) {
	return 0, nil
}

var _ statecontract.Client = (*fakeStateClient)(nil)
