package service

import (
	"context"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestServiceForwardsAccountOperations(t *testing.T) {
	stores := newFakeStores()
	svc := NewService(stores, stores, stores)
	account := &statecontract.Account{
		Username:     "alice",
		PasswordHash: "hash",
		PlayerID:     7,
	}

	if err := svc.CreateAccount(context.Background(), account); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	got, err := svc.GetAccount(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if got.Username != account.Username {
		t.Fatalf("username = %q, want %q", got.Username, account.Username)
	}
}

func TestServiceForwardsSessionOperations(t *testing.T) {
	stores := newFakeStores()
	svc := NewService(stores, stores, stores)
	session := &statecontract.Session{
		Token:     "token-1",
		PlayerID:  7,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := svc.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	got, err := svc.GetSession(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if got.Token != session.Token {
		t.Fatalf("token = %q, want %q", got.Token, session.Token)
	}
	if err := svc.DeleteSession(context.Background(), "token-1"); err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
	if _, err := svc.GetSession(context.Background(), "token-1"); err == nil {
		t.Fatalf("GetSession returned nil error, want missing session error")
	}
}

func TestServiceForwardsPlayerOperations(t *testing.T) {
	stores := newFakeStores()
	stores.nextPlayerID = 7
	svc := NewService(stores, stores, stores)

	id, err := svc.NextPlayerID(context.Background())
	if err != nil {
		t.Fatalf("NextPlayerID returned error: %v", err)
	}
	if id != 7 {
		t.Fatalf("id = %d, want 7", id)
	}

	player := &statecontract.Player{
		ID:       id,
		Nickname: "Alice",
	}
	if err := svc.CreatePlayer(context.Background(), player); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	got, err := svc.GetPlayer(context.Background(), id)
	if err != nil {
		t.Fatalf("GetPlayer returned error: %v", err)
	}
	if got.Nickname != player.Nickname {
		t.Fatalf("nickname = %q, want %q", got.Nickname, player.Nickname)
	}
}

func TestServiceRegisterAccountCreatesAccountPlayerAndSession(t *testing.T) {
	stores := newFakeStores()
	stores.nextPlayerID = 7
	svc := NewService(stores, stores, stores)
	expiresAt := time.Now().Add(time.Hour)

	result, err := svc.RegisterAccount(context.Background(), statecontract.RegisterAccountInput{
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
	if result.Account.PlayerID != 7 {
		t.Fatalf("account player id = %d, want 7", result.Account.PlayerID)
	}
	if result.Player.ID != 7 {
		t.Fatalf("player id = %d, want 7", result.Player.ID)
	}
	if result.Player.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want Alice", result.Player.Nickname)
	}
	if result.Session.Token != "token-1" {
		t.Fatalf("session token = %q, want token-1", result.Session.Token)
	}
	if !result.Session.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("session expires at = %v, want %v", result.Session.ExpiresAt, expiresAt)
	}
}

func TestServiceRegisterAccountExistingAccountDoesNotCreatePlayerOrSession(t *testing.T) {
	stores := newFakeStores()
	stores.nextPlayerID = 7
	stores.accounts["alice"] = &statecontract.Account{
		Username:     "alice",
		PasswordHash: "old-hash",
		PlayerID:     1,
	}
	svc := NewService(stores, stores, stores)

	_, err := svc.RegisterAccount(context.Background(), statecontract.RegisterAccountInput{
		Username:         "alice",
		PasswordHash:     "hash",
		Nickname:         "Alice",
		SessionToken:     "token-1",
		SessionExpiresAt: time.Now().Add(time.Hour),
	})
	if err != statecontract.ErrAccountExists {
		t.Fatalf("RegisterAccount error = %v, want %v", err, statecontract.ErrAccountExists)
	}
	if len(stores.players) != 0 {
		t.Fatalf("players = %d, want 0", len(stores.players))
	}
	if len(stores.sessions) != 0 {
		t.Fatalf("sessions = %d, want 0", len(stores.sessions))
	}
}

type fakeStores struct {
	accounts     map[string]*statecontract.Account
	sessions     map[string]*statecontract.Session
	players      map[int64]*statecontract.Player
	nextPlayerID int64
}

func newFakeStores() *fakeStores {
	return &fakeStores{
		accounts: make(map[string]*statecontract.Account),
		sessions: make(map[string]*statecontract.Session),
		players:  make(map[int64]*statecontract.Player),
	}
}

func (f *fakeStores) CreateAccount(_ context.Context, account *statecontract.Account) error {
	cp := *account
	f.accounts[account.Username] = &cp
	return nil
}

func (f *fakeStores) GetAccount(_ context.Context, username string) (*statecontract.Account, error) {
	account, ok := f.accounts[username]
	if !ok {
		return nil, statecontract.ErrAccountNotFound
	}
	cp := *account
	return &cp, nil
}

func (f *fakeStores) CreateSession(_ context.Context, session *statecontract.Session) error {
	cp := *session
	f.sessions[session.Token] = &cp
	return nil
}

func (f *fakeStores) GetSession(_ context.Context, token string) (*statecontract.Session, error) {
	session, ok := f.sessions[token]
	if !ok {
		return nil, statecontract.ErrSessionNotFound
	}
	cp := *session
	return &cp, nil
}

func (f *fakeStores) DeleteSession(_ context.Context, token string) error {
	delete(f.sessions, token)
	return nil
}

func (f *fakeStores) CreatePlayer(_ context.Context, player *statecontract.Player) error {
	cp := *player
	f.players[player.ID] = &cp
	return nil
}

func (f *fakeStores) GetPlayer(_ context.Context, id int64) (*statecontract.Player, error) {
	player, ok := f.players[id]
	if !ok {
		return nil, statecontract.ErrPlayerNotFound
	}
	cp := *player
	return &cp, nil
}

func (f *fakeStores) NextPlayerID(_ context.Context) (int64, error) {
	return f.nextPlayerID, nil
}

var _ statecontract.Client = (*Service)(nil)
