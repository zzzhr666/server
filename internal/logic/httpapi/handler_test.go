package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"server/internal/logic/auth"
	playerpkg "server/internal/logic/player"
	"strings"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestRegisterAuthHTTP(t *testing.T) {
	auths := newFakeAuthService()
	handler := newTestHandlerWithAuth(auths).Routes()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"username":"alice","password":"password123","nickname":"Alice","avatar":"alice.png"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp authSessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("token is empty")
	}
	if resp.Player.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want %q", resp.Player.Nickname, "Alice")
	}
}

func TestRegisterAuthHTTPInvalidJSON(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"username":`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestLoginAuthHTTP(t *testing.T) {
	auths := newFakeAuthService()
	auths.accounts["alice"] = &playerpkg.Player{ID: 7, Nickname: "Alice", Avatar: "alice.png"}
	handler := newTestHandlerWithAuth(auths).Routes()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice","password":"password123"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp authSessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Player.ID != 7 {
		t.Fatalf("player id = %d, want 7", resp.Player.ID)
	}
}

func TestLoginAuthHTTPInvalidCredentials(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice","password":"wrong"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestLogoutAuthHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	handler := newTestHandlerWithAuth(auths).Routes()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, ok := auths.sessions[session.Token]; ok {
		t.Fatalf("session token was not deleted")
	}
}

func TestLogoutAuthHTTPMissingToken(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestMeAuthHTTP(t *testing.T) {
	auths := newFakeAuthService()
	player := &playerpkg.Player{ID: 7, Nickname: "Alice", Avatar: "alice.png"}
	auths.players[7] = player
	session := auths.newSession(7)
	handler := newTestHandlerWithAuth(auths).Routes()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp playerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != 7 {
		t.Fatalf("player id = %d, want 7", resp.ID)
	}
	if resp.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want %q", resp.Nickname, "Alice")
	}
}

func TestMeAuthHTTPInvalidToken(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func newTestHandler() *Handler {
	return newTestHandlerWithAuth(newFakeAuthService())
}

func newTestHandlerWithAuth(auths *fakeAuthService) *Handler {
	return NewHandler(HandlerConfig{AuthService: auths})
}

type fakeAuthService struct {
	nextPlayerID int64
	accounts     map[string]*playerpkg.Player
	players      map[int64]*playerpkg.Player
	sessions     map[string]*auth.Session
}

func newFakeAuthService() *fakeAuthService {
	return &fakeAuthService{
		nextPlayerID: 1,
		accounts:     make(map[string]*playerpkg.Player),
		players:      make(map[int64]*playerpkg.Player),
		sessions:     make(map[string]*auth.Session),
	}
}

func (s *fakeAuthService) Register(ctx context.Context, input auth.RegisterInput) (*auth.AuthorizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Username == "" {
		return nil, auth.ErrInvalidUsername
	}
	if input.PlainPassword == "" {
		return nil, auth.ErrInvalidPassword
	}
	if input.Nickname == "" {
		return nil, playerpkg.ErrInvalidNickname
	}
	if _, exists := s.accounts[input.Username]; exists {
		return nil, auth.ErrAccountExists
	}
	player := &playerpkg.Player{
		ID:       s.nextPlayerID,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}
	s.nextPlayerID++
	s.accounts[input.Username] = player
	s.players[player.ID] = player
	session := s.newSession(player.ID)
	return &auth.AuthorizeResult{Session: session, Player: clonePlayer(player)}, nil
}

func (s *fakeAuthService) Login(ctx context.Context, input auth.LoginInput) (*auth.AuthorizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Username == "" {
		return nil, auth.ErrInvalidUsername
	}
	if input.PlainPassword == "" {
		return nil, auth.ErrInvalidPassword
	}
	player, ok := s.accounts[input.Username]
	if !ok || input.PlainPassword != "password123" {
		return nil, auth.ErrInvalidCredentials
	}
	session := s.newSession(player.ID)
	return &auth.AuthorizeResult{Session: session, Player: clonePlayer(player)}, nil
}

func (s *fakeAuthService) Logout(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return auth.ErrSessionNotFound
	}
	delete(s.sessions, token)
	return nil
}

func (s *fakeAuthService) GetCurrentPlayer(ctx context.Context, token string) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session, err := s.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	player, ok := s.players[session.PlayerID]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	return clonePlayer(player), nil
}

func (s *fakeAuthService) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session, ok := s.sessions[token]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	cp := *session
	return &cp, nil
}

func (s *fakeAuthService) newSession(playerID int64) *auth.Session {
	token := "token-" + time.Now().Format("150405.000000000")
	session := &auth.Session{
		Token:     token,
		PlayerID:  playerID,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.sessions[token] = session
	return session
}

func clonePlayer(player *playerpkg.Player) *playerpkg.Player {
	if player == nil {
		return nil
	}
	cp := *player
	return &cp
}

var _ auth.Service = (*fakeAuthService)(nil)
