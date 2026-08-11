package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/logic/auth"
	"server/internal/logic/player"
)

func TestHealth(t *testing.T) {
	handler := NewHandler(HandlerConfig{AuthService: &fakePublicAuthService{}, ServerName: "logic-test"}).Routes()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok server_name = logic-test" {
		t.Fatalf("health response = (%d, %q), want (200, %q)", recorder.Code, recorder.Body.String(), "ok server_name = logic-test")
	}
}

func TestRegisterAuthHTTP(t *testing.T) {
	authService := &fakePublicAuthService{authorizeResult: testAuthorizeResult()}
	handler := NewHandler(HandlerConfig{AuthService: authService}).Routes()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"username":"alice","password":"password123","nickname":"Alice","avatar":"alice.png"}`))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var response authSessionResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Token != "session-token" || response.Player.ID != 7 || authService.registerInput.Username != "alice" || authService.registerInput.Nickname != "Alice" {
		t.Fatalf("register response/input = (%+v, %+v), want player 7 and alice input", response, authService.registerInput)
	}
}

func TestRegisterAuthHTTPRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(HandlerConfig{AuthService: &fakePublicAuthService{}}).Routes()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"username":`)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginAuthHTTP(t *testing.T) {
	authService := &fakePublicAuthService{authorizeResult: testAuthorizeResult()}
	handler := NewHandler(HandlerConfig{AuthService: authService}).Routes()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice","password":"password123"}`))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || authService.loginInput.Username != "alice" {
		t.Fatalf("login response/input = (%d, %+v), want status 200 and alice input", recorder.Code, authService.loginInput)
	}
}

func TestLoginAuthHTTPRejectsInvalidCredentials(t *testing.T) {
	authService := &fakePublicAuthService{err: auth.ErrInvalidCredentials}
	handler := NewHandler(HandlerConfig{AuthService: authService}).Routes()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice","password":"wrong"}`))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedHTTPRoutesAreRemoved(t *testing.T) {
	handler := NewHandler(HandlerConfig{AuthService: &fakePublicAuthService{}}).Routes()
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/auth/logout"},
		{http.MethodGet, "/auth/me"},
		{http.MethodGet, "/friends"},
		{http.MethodPost, "/friends/requests"},
		{http.MethodGet, "/growth"},
		{http.MethodPost, "/growth/upgrade"},
	}
	for _, item := range paths {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(item.method, item.path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want 404", item.method, item.path, recorder.Code)
		}
	}
}

func testAuthorizeResult() *auth.AuthorizeResult {
	return &auth.AuthorizeResult{
		Session: &auth.Session{Token: "session-token", PlayerID: 7},
		Player:  &player.Player{ID: 7, Nickname: "Alice", Avatar: "alice.png"},
	}
}

type fakePublicAuthService struct {
	authorizeResult *auth.AuthorizeResult
	err             error
	registerInput   auth.RegisterInput
	loginInput      auth.LoginInput
}

func (f *fakePublicAuthService) Register(_ context.Context, input auth.RegisterInput) (*auth.AuthorizeResult, error) {
	f.registerInput = input
	return f.authorizeResult, f.err
}

func (f *fakePublicAuthService) Login(_ context.Context, input auth.LoginInput) (*auth.AuthorizeResult, error) {
	f.loginInput = input
	return f.authorizeResult, f.err
}

func (f *fakePublicAuthService) Logout(context.Context, string) error {
	return errors.New("unexpected Logout call")
}

func (f *fakePublicAuthService) GetSession(context.Context, string) (*auth.Session, error) {
	return nil, errors.New("unexpected GetSession call")
}

func (f *fakePublicAuthService) GetCurrentPlayer(context.Context, string) (*player.Player, error) {
	return nil, errors.New("unexpected GetCurrentPlayer call")
}

var _ auth.Service = (*fakePublicAuthService)(nil)
