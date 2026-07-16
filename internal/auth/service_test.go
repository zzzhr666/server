package auth

import (
	"context"
	"errors"
	playerpkg "learning/internal/player"
	"testing"
	"time"
)

func TestRegisterCreatesAccountPlayerAndSession(t *testing.T) {
	repo := newFakeAuthRepository()
	players := newFakePlayerService()
	service := NewService(repo, players, time.Hour)

	session, err := service.Register(context.Background(), RegisterInput{
		Username:      "alice",
		PlainPassword: "password123",
		Name:          "Alice",
		Avatar:        "alice.png",
		Email:         "alice@example.com",
		Phone:         "13800000000",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if session.Token == "" {
		t.Fatalf("session token is empty")
	}
	if session.PlayerID != 1 {
		t.Fatalf("session player id = %d, want 1", session.PlayerID)
	}
	if !session.ExpiresAt.After(time.Now()) {
		t.Fatalf("session expires_at = %v, want future time", session.ExpiresAt)
	}

	account, err := repo.GetAccount(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if account.PasswordHash == "password123" {
		t.Fatalf("password was stored in plaintext")
	}
	if !checkPassword(account.PasswordHash, "password123") {
		t.Fatalf("stored password hash does not match original password")
	}
	if account.PlayerID != 1 {
		t.Fatalf("account player id = %d, want 1", account.PlayerID)
	}

	player, err := players.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("Get player returned error: %v", err)
	}
	if player.Name != "Alice" || player.Avatar != "alice.png" || player.Email != "alice@example.com" || player.Phone != "13800000000" {
		t.Fatalf("player profile = %#v, want register input profile", player)
	}
}

func TestRegisterDuplicateAccount(t *testing.T) {
	repo := newFakeAuthRepository()
	players := newFakePlayerService()
	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if err := repo.CreateAccount(context.Background(), &Account{
		Username:     "alice",
		PasswordHash: hash,
		PlayerID:     99,
	}); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	service := NewService(repo, players, time.Hour)

	_, err = service.Register(context.Background(), RegisterInput{
		Username:      "alice",
		PlainPassword: "new-password",
		Name:          "Alice",
	})
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("Register error = %v, want %v", err, ErrAccountExists)
	}
	if players.created != 0 {
		t.Fatalf("created players = %d, want 0", players.created)
	}
}

func TestLoginCreatesSession(t *testing.T) {
	repo := newFakeAuthRepository()
	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if err := repo.CreateAccount(context.Background(), &Account{
		Username:     "alice",
		PasswordHash: hash,
		PlayerID:     7,
	}); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	service := NewService(repo, newFakePlayerService(), time.Hour)

	session, err := service.Login(context.Background(), LoginInput{
		Username:      "alice",
		PlainPassword: "password123",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if session.Token == "" {
		t.Fatalf("session token is empty")
	}
	if session.PlayerID != 7 {
		t.Fatalf("session player id = %d, want 7", session.PlayerID)
	}

	stored, err := repo.GetSession(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if stored.PlayerID != 7 {
		t.Fatalf("stored session player id = %d, want 7", stored.PlayerID)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	repo := newFakeAuthRepository()
	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if err := repo.CreateAccount(context.Background(), &Account{
		Username:     "alice",
		PasswordHash: hash,
		PlayerID:     7,
	}); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	service := NewService(repo, newFakePlayerService(), time.Hour)

	_, err = service.Login(context.Background(), LoginInput{
		Username:      "alice",
		PlainPassword: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want %v", err, ErrInvalidCredentials)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("stored sessions = %d, want 0", len(repo.sessions))
	}
}

type fakeAuthRepository struct {
	accounts map[string]*Account
	sessions map[string]*Session
}

func newFakeAuthRepository() *fakeAuthRepository {
	return &fakeAuthRepository{
		accounts: make(map[string]*Account),
		sessions: make(map[string]*Session),
	}
}

func (f *fakeAuthRepository) CreateAccount(_ context.Context, account *Account) error {
	if _, ok := f.accounts[account.Username]; ok {
		return ErrAccountExists
	}
	cp := *account
	f.accounts[account.Username] = &cp
	return nil
}

func (f *fakeAuthRepository) GetAccount(_ context.Context, username string) (*Account, error) {
	account, ok := f.accounts[username]
	if !ok {
		return nil, ErrAccountNotFound
	}
	cp := *account
	return &cp, nil
}

func (f *fakeAuthRepository) CreateSession(_ context.Context, session *Session) error {
	cp := *session
	f.sessions[session.Token] = &cp
	return nil
}

func (f *fakeAuthRepository) GetSession(_ context.Context, token string) (*Session, error) {
	session, ok := f.sessions[token]
	if !ok || time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionNotFound
	}
	cp := *session
	return &cp, nil
}

func (f *fakeAuthRepository) DeleteSession(_ context.Context, token string) error {
	delete(f.sessions, token)
	return nil
}

type fakePlayerService struct {
	nextID  int64
	created int
	players map[int64]*playerpkg.Player
}

func newFakePlayerService() *fakePlayerService {
	return &fakePlayerService{
		nextID:  1,
		players: make(map[int64]*playerpkg.Player),
	}
}

func (f *fakePlayerService) Create(_ context.Context, input playerpkg.CreateInput) (*playerpkg.Player, error) {
	if input.Name == "" {
		return nil, playerpkg.ErrInvalidName
	}
	player := &playerpkg.Player{
		ID:     f.nextID,
		Name:   input.Name,
		Avatar: input.Avatar,
		Email:  input.Email,
		Phone:  input.Phone,
	}
	f.nextID++
	f.created++
	f.players[player.ID] = player
	return player, nil
}

func (f *fakePlayerService) Get(_ context.Context, id int64) (*playerpkg.Player, error) {
	player, ok := f.players[id]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	cp := *player
	return &cp, nil
}

func (f *fakePlayerService) UpdateProfile(_ context.Context, id int64, input playerpkg.UpdateProfileInput) (*playerpkg.Player, error) {
	player, ok := f.players[id]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	if input.Name != nil {
		player.Name = *input.Name
	}
	if input.Avatar != nil {
		player.Avatar = *input.Avatar
	}
	if input.Email != nil {
		player.Email = *input.Email
	}
	if input.Phone != nil {
		player.Phone = *input.Phone
	}
	cp := *player
	return &cp, nil
}
