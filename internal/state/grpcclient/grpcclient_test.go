package grpcclient

import (
	"context"
	"errors"
	"testing"
	"time"

	statecontract "server/internal/contract/state"
	"server/internal/contract/statepb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMapGRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "nil",
			err:  nil,
			want: nil,
		},
		{
			name: "account exists",
			err:  status.Error(codes.AlreadyExists, statecontract.ErrAccountExists.Error()),
			want: statecontract.ErrAccountExists,
		},
		{
			name: "account not found",
			err:  status.Error(codes.NotFound, statecontract.ErrAccountNotFound.Error()),
			want: statecontract.ErrAccountNotFound,
		},
		{
			name: "session not found",
			err:  status.Error(codes.NotFound, statecontract.ErrSessionNotFound.Error()),
			want: statecontract.ErrSessionNotFound,
		},
		{
			name: "player not found",
			err:  status.Error(codes.NotFound, statecontract.ErrPlayerNotFound.Error()),
			want: statecontract.ErrPlayerNotFound,
		},
		{
			name: "presence not found",
			err:  status.Error(codes.NotFound, statecontract.ErrPresenceNotFound.Error()),
			want: statecontract.ErrPresenceNotFound,
		},
		{
			name: "invalid presence",
			err:  status.Error(codes.InvalidArgument, statecontract.ErrInvalidPresence.Error()),
			want: statecontract.ErrInvalidPresence,
		},
		{
			name: "unknown already exists message",
			err:  status.Error(codes.AlreadyExists, "room already exists"),
			want: status.Error(codes.AlreadyExists, "room already exists"),
		},
		{
			name: "unknown not found message",
			err:  status.Error(codes.NotFound, "room not found"),
			want: status.Error(codes.NotFound, "room not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapGRPCError(tt.err)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("mapGRPCError returned %v, want nil", got)
				}
				return
			}
			if errors.Is(got, tt.want) {
				return
			}
			if got.Error() != tt.want.Error() {
				t.Fatalf("mapGRPCError returned %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientGetAccount(t *testing.T) {
	grpcState := &fakeStateServiceClient{
		account: &statepb.Account{
			Username:     "alice",
			PasswordHash: "hash",
			PlayerId:     7,
		},
	}
	client := NewClient(grpcState)

	account, err := client.GetAccount(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if grpcState.gotUsername != "alice" {
		t.Fatalf("grpc got username = %q, want alice", grpcState.gotUsername)
	}
	if account.PlayerID != 7 {
		t.Fatalf("player id = %d, want 7", account.PlayerID)
	}
}

func TestClientGetAccountMapsNotFound(t *testing.T) {
	client := NewClient(&fakeStateServiceClient{
		err: status.Error(codes.NotFound, statecontract.ErrAccountNotFound.Error()),
	})

	_, err := client.GetAccount(context.Background(), "missing")
	if !errors.Is(err, statecontract.ErrAccountNotFound) {
		t.Fatalf("GetAccount error = %v, want %v", err, statecontract.ErrAccountNotFound)
	}
}

func TestClientRegisterAccount(t *testing.T) {
	expiresAt := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	grpcState := &fakeStateServiceClient{
		registerResponse: &statepb.RegisterAccountResponse{
			Account: &statepb.Account{Username: "alice", PasswordHash: "hash", PlayerId: 7},
			Player:  &statepb.Player{Id: 7, Nickname: "Alice", Avatar: "avatar", Email: "a@example.com", Phone: "123"},
			Session: &statepb.Session{Token: "token-1", PlayerId: 7, ExpiresAt: timestamppb.New(expiresAt)},
		},
	}
	client := NewClient(grpcState)

	result, err := client.RegisterAccount(context.Background(), statecontract.RegisterAccountInput{
		Username:         "alice",
		PasswordHash:     "hash",
		Nickname:         "Alice",
		Avatar:           "avatar",
		Email:            "a@example.com",
		Phone:            "123",
		SessionToken:     "token-1",
		SessionExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("RegisterAccount returned error: %v", err)
	}
	if grpcState.registerRequest.GetUsername() != "alice" {
		t.Fatalf("register username = %q, want alice", grpcState.registerRequest.GetUsername())
	}
	if !grpcState.registerRequest.GetSessionExpiresAt().AsTime().Equal(expiresAt) {
		t.Fatalf("register expires at = %v, want %v", grpcState.registerRequest.GetSessionExpiresAt().AsTime(), expiresAt)
	}
	if result.Player.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want Alice", result.Player.Nickname)
	}
	if !result.Session.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("session expires at = %v, want %v", result.Session.ExpiresAt, expiresAt)
	}
}

func TestClientSessionMethods(t *testing.T) {
	expiresAt := time.Date(2026, 7, 19, 13, 0, 0, 0, time.UTC)
	grpcState := &fakeStateServiceClient{
		session: &statepb.Session{Token: "token-1", PlayerId: 7, ExpiresAt: timestamppb.New(expiresAt)},
	}
	client := NewClient(grpcState)

	err := client.CreateSession(context.Background(), &statecontract.Session{Token: "token-1", PlayerID: 7, ExpiresAt: expiresAt})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if grpcState.createdSession.GetToken() != "token-1" {
		t.Fatalf("created session token = %q, want token-1", grpcState.createdSession.GetToken())
	}

	session, err := client.GetSession(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if session.PlayerID != 7 {
		t.Fatalf("session player id = %d, want 7", session.PlayerID)
	}

	if err := client.DeleteSession(context.Background(), "token-1"); err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
	if grpcState.deletedToken != "token-1" {
		t.Fatalf("deleted token = %q, want token-1", grpcState.deletedToken)
	}
}

func TestClientPlayerMethods(t *testing.T) {
	grpcState := &fakeStateServiceClient{
		player:       &statepb.Player{Id: 7, Nickname: "Alice"},
		nextPlayerID: 8,
	}
	client := NewClient(grpcState)

	if err := client.CreatePlayer(context.Background(), &statecontract.Player{ID: 7, Nickname: "Alice"}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	if grpcState.createdPlayer.GetId() != 7 {
		t.Fatalf("created player id = %d, want 7", grpcState.createdPlayer.GetId())
	}

	player, err := client.GetPlayer(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetPlayer returned error: %v", err)
	}
	if player.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want Alice", player.Nickname)
	}

	id, err := client.NextPlayerID(context.Background())
	if err != nil {
		t.Fatalf("NextPlayerID returned error: %v", err)
	}
	if id != 8 {
		t.Fatalf("next player id = %d, want 8", id)
	}
}

func TestClientCreateAccount(t *testing.T) {
	grpcState := &fakeStateServiceClient{}
	client := NewClient(grpcState)

	err := client.CreateAccount(context.Background(), &statecontract.Account{
		Username:     "alice",
		PasswordHash: "hash",
		PlayerID:     7,
	})
	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if grpcState.createdAccount.GetUsername() != "alice" {
		t.Fatalf("created account username = %q, want alice", grpcState.createdAccount.GetUsername())
	}
}

func TestClientPresenceMethods(t *testing.T) {
	updatedAt := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	grpcState := &fakeStateServiceClient{
		presence: &statepb.Presence{
			PlayerId:   7,
			ServerName: "logic-1",
			Status:     "online",
			UpdatedAt:  timestamppb.New(updatedAt),
		},
	}
	client := NewClient(grpcState)

	err := client.SetPresence(context.Background(), &statecontract.Presence{
		PlayerID:   7,
		ServerName: "logic-1",
		Status:     "online",
		UpdatedAt:  updatedAt,
	}, time.Minute)
	if err != nil {
		t.Fatalf("SetPresence returned error: %v", err)
	}
	if grpcState.setPresence.GetPlayerId() != 7 {
		t.Fatalf("set presence player id = %d, want 7", grpcState.setPresence.GetPlayerId())
	}
	if grpcState.setPresence.GetStatus() != "online" {
		t.Fatalf("set presence status = %q, want online", grpcState.setPresence.GetStatus())
	}
	if grpcState.setPresenceTTL != time.Minute {
		t.Fatalf("set presence ttl = %v, want %v", grpcState.setPresenceTTL, time.Minute)
	}

	presence, err := client.GetPresence(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetPresence returned error: %v", err)
	}
	if presence.ServerName != "logic-1" {
		t.Fatalf("presence server name = %q, want logic-1", presence.ServerName)
	}

	if err := client.ClearPresence(context.Background(), 7, "logic-1"); err != nil {
		t.Fatalf("ClearPresence returned error: %v", err)
	}
	if grpcState.clearedPlayerID != 7 {
		t.Fatalf("cleared player id = %d, want 7", grpcState.clearedPlayerID)
	}
	if grpcState.clearedServerName != "logic-1" {
		t.Fatalf("cleared server name = %q, want logic-1", grpcState.clearedServerName)
	}
}

type fakeStateServiceClient struct {
	statepb.UnimplementedStateServiceServer

	account           *statepb.Account
	createdAccount    *statepb.Account
	gotUsername       string
	registerRequest   *statepb.RegisterAccountRequest
	registerResponse  *statepb.RegisterAccountResponse
	session           *statepb.Session
	createdSession    *statepb.Session
	deletedToken      string
	player            *statepb.Player
	createdPlayer     *statepb.Player
	nextPlayerID      int64
	presence          *statepb.Presence
	setPresence       *statepb.Presence
	setPresenceTTL    time.Duration
	clearedPlayerID   int64
	clearedServerName string
	err               error
}

func (f *fakeStateServiceClient) CreateAccount(_ context.Context, in *statepb.CreateAccountRequest, _ ...grpc.CallOption) (*statepb.CreateAccountResponse, error) {
	f.createdAccount = in.GetAccount()
	return &statepb.CreateAccountResponse{}, f.err
}

func (f *fakeStateServiceClient) GetAccount(_ context.Context, in *statepb.GetAccountRequest, _ ...grpc.CallOption) (*statepb.GetAccountResponse, error) {
	f.gotUsername = in.GetUsername()
	if f.err != nil {
		return nil, f.err
	}
	return &statepb.GetAccountResponse{Account: f.account}, nil
}

func (f *fakeStateServiceClient) RegisterAccount(_ context.Context, in *statepb.RegisterAccountRequest, _ ...grpc.CallOption) (*statepb.RegisterAccountResponse, error) {
	f.registerRequest = in
	if f.err != nil {
		return nil, f.err
	}
	return f.registerResponse, nil
}

func (f *fakeStateServiceClient) CreateSession(_ context.Context, in *statepb.CreateSessionRequest, _ ...grpc.CallOption) (*statepb.CreateSessionResponse, error) {
	f.createdSession = in.GetSession()
	return &statepb.CreateSessionResponse{}, f.err
}

func (f *fakeStateServiceClient) GetSession(_ context.Context, in *statepb.GetSessionRequest, _ ...grpc.CallOption) (*statepb.GetSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statepb.GetSessionResponse{Session: f.session}, nil
}

func (f *fakeStateServiceClient) DeleteSession(_ context.Context, in *statepb.DeleteSessionRequest, _ ...grpc.CallOption) (*statepb.DeleteSessionResponse, error) {
	f.deletedToken = in.GetToken()
	return &statepb.DeleteSessionResponse{}, f.err
}

func (f *fakeStateServiceClient) CreatePlayer(_ context.Context, in *statepb.CreatePlayerRequest, _ ...grpc.CallOption) (*statepb.CreatePlayerResponse, error) {
	f.createdPlayer = in.GetPlayer()
	return &statepb.CreatePlayerResponse{}, f.err
}

func (f *fakeStateServiceClient) GetPlayer(_ context.Context, in *statepb.GetPlayerRequest, _ ...grpc.CallOption) (*statepb.GetPlayerResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statepb.GetPlayerResponse{Player: f.player}, nil
}

func (f *fakeStateServiceClient) NextPlayerID(context.Context, *statepb.NextPlayerIDRequest, ...grpc.CallOption) (*statepb.NextPlayerIDResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statepb.NextPlayerIDResponse{Id: f.nextPlayerID}, nil
}

func (f *fakeStateServiceClient) SetPresence(_ context.Context, in *statepb.SetPresenceRequest, _ ...grpc.CallOption) (*statepb.SetPresenceResponse, error) {
	f.setPresence = in.GetPresence()
	f.setPresenceTTL = in.GetTtl().AsDuration()
	return &statepb.SetPresenceResponse{}, f.err
}

func (f *fakeStateServiceClient) GetPresence(_ context.Context, _ *statepb.GetPresenceRequest, _ ...grpc.CallOption) (*statepb.GetPresenceResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statepb.GetPresenceResponse{Presence: f.presence}, nil
}

func (f *fakeStateServiceClient) ClearPresence(_ context.Context, in *statepb.ClearPresenceRequest, _ ...grpc.CallOption) (*statepb.ClearPresenceResponse, error) {
	f.clearedPlayerID = in.GetPlayerId()
	f.clearedServerName = in.GetServerName()
	return &statepb.ClearPresenceResponse{}, f.err
}

var _ statepb.StateServiceClient = (*fakeStateServiceClient)(nil)
