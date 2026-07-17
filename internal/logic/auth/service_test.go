package auth

import (
	"context"
	"errors"
	playerpkg "server/internal/logic/player"
	"testing"
	"time"
)

func TestRegisterCreatesAccountPlayerAndSession(t *testing.T) {
	repo := newFakeAuthRepository()
	players := newFakePlayerService()
	service := NewService(repo, players, time.Hour)

	result, err := service.Register(context.Background(), RegisterInput{
		Username:      "alice",
		PlainPassword: "password123",
		Nickname:      "Alice",
		Avatar:        "alice.png",
		Email:         "alice@example.com",
		Phone:         "13800000000",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if result.Session.Token == "" {
		t.Fatalf("session token is empty")
	}
	if result.Session.PlayerID != 1 {
		t.Fatalf("session player id = %d, want 1", result.Session.PlayerID)
	}
	if !result.Session.ExpiresAt.After(time.Now()) {
		t.Fatalf("session expires_at = %v, want future time", result.Session.ExpiresAt)
	}
	if result.Player == nil {
		t.Fatalf("result player is nil")
	}
	if result.Player.ID != 1 {
		t.Fatalf("result player id = %d, want 1", result.Player.ID)
	}
	if result.Player.Nickname != "Alice" || result.Player.Avatar != "alice.png" || result.Player.Email != "alice@example.com" || result.Player.Phone != "13800000000" {
		t.Fatalf("result player profile = %#v, want register input profile", result.Player)
	}
	if repo.registerCalls != 1 {
		t.Fatalf("register calls = %d, want 1", repo.registerCalls)
	}
	if players.created != 0 {
		t.Fatalf("created players through player service = %d, want 0", players.created)
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

	player, ok := repo.players[1]
	if !ok {
		t.Fatalf("registered player was not stored")
	}
	if player.Nickname != "Alice" || player.Avatar != "alice.png" || player.Email != "alice@example.com" || player.Phone != "13800000000" {
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
		Nickname:      "Alice",
	})
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("Register error = %v, want %v", err, ErrAccountExists)
	}
	if players.created != 0 {
		t.Fatalf("created players = %d, want 0", players.created)
	}
	if len(repo.players) != 0 {
		t.Fatalf("stored players = %d, want 0", len(repo.players))
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("stored sessions = %d, want 0", len(repo.sessions))
	}
}

func TestLoginCreatesSession(t *testing.T) {
	repo := newFakeAuthRepository()
	players := newFakePlayerService()
	players.players[7] = &playerpkg.Player{
		ID:       7,
		Nickname: "Alice",
		Avatar:   "alice.png",
	}
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
	service := NewService(repo, players, time.Hour)

	result, err := service.Login(context.Background(), LoginInput{
		Username:      "alice",
		PlainPassword: "password123",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.Session.Token == "" {
		t.Fatalf("session token is empty")
	}
	if result.Session.PlayerID != 7 {
		t.Fatalf("session player id = %d, want 7", result.Session.PlayerID)
	}
	if result.Player == nil {
		t.Fatalf("result player is nil")
	}
	if result.Player.ID != 7 {
		t.Fatalf("result player id = %d, want 7", result.Player.ID)
	}

	stored, err := repo.GetSession(context.Background(), result.Session.Token)
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

func TestLoginMissingPlayerDoesNotCreateSession(t *testing.T) {
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
		PlainPassword: "password123",
	})
	if !errors.Is(err, playerpkg.ErrNotFound) {
		t.Fatalf("Login error = %v, want %v", err, playerpkg.ErrNotFound)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("stored sessions = %d, want 0", len(repo.sessions))
	}
}

func TestGetCurrentPlayer(t *testing.T) {
	repo := newFakeAuthRepository()
	players := newFakePlayerService()
	players.players[7] = &playerpkg.Player{
		ID:       7,
		Nickname: "Alice",
		Avatar:   "alice.png",
	}
	service := NewService(repo, players, time.Hour)
	session := &Session{
		Token:     "token-1",
		PlayerID:  7,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	p, err := service.GetCurrentPlayer(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("GetCurrentPlayer returned error: %v", err)
	}
	if p.ID != 7 {
		t.Fatalf("player id = %d, want 7", p.ID)
	}
	if p.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want %q", p.Nickname, "Alice")
	}
}

func TestGetCurrentPlayerInvalidToken(t *testing.T) {
	service := NewService(newFakeAuthRepository(), newFakePlayerService(), time.Hour)

	_, err := service.GetCurrentPlayer(context.Background(), "missing-token")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetCurrentPlayer error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestGetCurrentPlayerMissingPlayer(t *testing.T) {
	repo := newFakeAuthRepository()
	service := NewService(repo, newFakePlayerService(), time.Hour)
	session := &Session{
		Token:     "token-1",
		PlayerID:  7,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	_, err := service.GetCurrentPlayer(context.Background(), "token-1")
	if !errors.Is(err, playerpkg.ErrNotFound) {
		t.Fatalf("GetCurrentPlayer error = %v, want %v", err, playerpkg.ErrNotFound)
	}
}

type fakeAuthRepository struct {
	nextPlayerID  int64
	registerCalls int
	accounts      map[string]*Account
	sessions      map[string]*Session
	players       map[int64]*playerpkg.Player
}

func newFakeAuthRepository() *fakeAuthRepository {
	return &fakeAuthRepository{
		nextPlayerID: 1,
		accounts:     make(map[string]*Account),
		sessions:     make(map[string]*Session),
		players:      make(map[int64]*playerpkg.Player),
	}
}

func (f *fakeAuthRepository) RegisterAccount(_ context.Context, input RegisterAccountInput) (*RegisterAccountResult, error) {
	f.registerCalls++
	if _, ok := f.accounts[input.Username]; ok {
		return nil, ErrAccountExists
	}
	player := &playerpkg.Player{
		ID:       f.nextPlayerID,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}
	f.nextPlayerID++
	account := &Account{
		Username:     input.Username,
		PasswordHash: input.PasswordHash,
		PlayerID:     player.ID,
	}
	session := &Session{
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

	return &RegisterAccountResult{
		Account: account,
		Player:  player,
		Session: session,
	}, nil
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
	if input.Nickname == "" {
		return nil, playerpkg.ErrInvalidNickname
	}
	player := &playerpkg.Player{
		ID:       f.nextID,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
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
