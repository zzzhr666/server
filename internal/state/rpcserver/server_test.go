package rpcserver

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
)

func TestGetAccount(t *testing.T) {
	state := &fakeStateClient{
		account: &statecontract.Account{
			Username:     "alice",
			PasswordHash: "hash",
			PlayerID:     7,
		},
	}
	server := NewServer(state)
	var reply GetAccountReply

	if err := server.GetAccount(GetAccountArgs{Username: "alice"}, &reply); err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if reply.Account.Username != "alice" {
		t.Fatalf("username = %q, want alice", reply.Account.Username)
	}
	if state.gotUsername != "alice" {
		t.Fatalf("state got username = %q, want alice", state.gotUsername)
	}
}

func TestGetAccountError(t *testing.T) {
	server := NewServer(&fakeStateClient{err: statecontract.ErrAccountNotFound})
	var reply GetAccountReply

	err := server.GetAccount(GetAccountArgs{Username: "missing"}, &reply)
	if !errors.Is(err, statecontract.ErrAccountNotFound) {
		t.Fatalf("GetAccount error = %v, want %v", err, statecontract.ErrAccountNotFound)
	}
}

func TestCreateAccount(t *testing.T) {
	state := &fakeStateClient{}
	server := NewServer(state)
	account := &statecontract.Account{
		Username:     "alice",
		PasswordHash: "hash",
		PlayerID:     7,
	}
	var reply CreateAccountReply

	if err := server.CreateAccount(CreateAccountArgs{Account: account}, &reply); err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if state.createdAccount == nil {
		t.Fatalf("created account is nil")
	}
	if state.createdAccount.Username != "alice" {
		t.Fatalf("created username = %q, want alice", state.createdAccount.Username)
	}
}

func TestRegisterAccount(t *testing.T) {
	state := &fakeStateClient{
		registerResult: &statecontract.RegisterAccountResult{
			Account: &statecontract.Account{Username: "alice", PlayerID: 7},
			Player:  &statecontract.Player{ID: 7, Nickname: "Alice"},
			Session: &statecontract.Session{Token: "token-1", PlayerID: 7},
		},
	}
	server := NewServer(state)
	var reply RegisterAccountReply

	err := server.RegisterAccount(RegisterAccountArgs{Input: statecontract.RegisterAccountInput{
		Username: "alice",
		Nickname: "Alice",
	}}, &reply)
	if err != nil {
		t.Fatalf("RegisterAccount returned error: %v", err)
	}
	if reply.Result.Player.ID != 7 {
		t.Fatalf("player id = %d, want 7", reply.Result.Player.ID)
	}
	if state.registerInput.Username != "alice" {
		t.Fatalf("register username = %q, want alice", state.registerInput.Username)
	}
}

func TestSessionMethods(t *testing.T) {
	state := &fakeStateClient{session: &statecontract.Session{Token: "token-1", PlayerID: 7}}
	server := NewServer(state)

	if err := server.CreateSession(CreateSessionArgs{Session: state.session}, &CreateSessionReply{}); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if state.createdSession.Token != "token-1" {
		t.Fatalf("created session token = %q, want token-1", state.createdSession.Token)
	}

	var getReply GetSessionReply
	if err := server.GetSession(GetSessionArgs{Token: "token-1"}, &getReply); err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if getReply.Session.PlayerID != 7 {
		t.Fatalf("session player id = %d, want 7", getReply.Session.PlayerID)
	}

	if err := server.DeleteSession(DeleteSessionArgs{Token: "token-1"}, &DeleteSessionReply{}); err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
	if state.deletedToken != "token-1" {
		t.Fatalf("deleted token = %q, want token-1", state.deletedToken)
	}
}

func TestPlayerMethods(t *testing.T) {
	state := &fakeStateClient{player: &statecontract.Player{ID: 7, Nickname: "Alice"}, nextPlayerID: 8}
	server := NewServer(state)

	if err := server.CreatePlayer(CreatePlayerArgs{Player: state.player}, &CreatePlayerReply{}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	if state.createdPlayer.ID != 7 {
		t.Fatalf("created player id = %d, want 7", state.createdPlayer.ID)
	}

	var getReply GetPlayerReply
	if err := server.GetPlayer(GetPlayerArgs{ID: 7}, &getReply); err != nil {
		t.Fatalf("GetPlayer returned error: %v", err)
	}
	if getReply.Player.Nickname != "Alice" {
		t.Fatalf("player nickname = %q, want Alice", getReply.Player.Nickname)
	}

	var nextReply NextPlayerIDReply
	if err := server.NextPlayerID(NextPlayerIDArgs{}, &nextReply); err != nil {
		t.Fatalf("NextPlayerID returned error: %v", err)
	}
	if nextReply.ID != 8 {
		t.Fatalf("next player id = %d, want 8", nextReply.ID)
	}
}

type fakeStateClient struct {
	account        *statecontract.Account
	createdAccount *statecontract.Account
	session        *statecontract.Session
	createdSession *statecontract.Session
	deletedToken   string
	player         *statecontract.Player
	createdPlayer  *statecontract.Player
	registerInput  statecontract.RegisterAccountInput
	registerResult *statecontract.RegisterAccountResult
	nextPlayerID   int64
	err            error
	gotUsername    string
}

func (f *fakeStateClient) CreateAccount(_ context.Context, account *statecontract.Account) error {
	f.createdAccount = account
	return nil
}

func (f *fakeStateClient) GetAccount(_ context.Context, username string) (*statecontract.Account, error) {
	f.gotUsername = username
	if f.err != nil {
		return nil, f.err
	}
	return f.account, nil
}

func (f *fakeStateClient) RegisterAccount(_ context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	f.registerInput = input
	return f.registerResult, nil
}

func (f *fakeStateClient) CreateSession(_ context.Context, session *statecontract.Session) error {
	f.createdSession = session
	return nil
}

func (f *fakeStateClient) GetSession(_ context.Context, _ string) (*statecontract.Session, error) {
	return f.session, nil
}

func (f *fakeStateClient) DeleteSession(_ context.Context, token string) error {
	f.deletedToken = token
	return nil
}

func (f *fakeStateClient) CreatePlayer(_ context.Context, player *statecontract.Player) error {
	f.createdPlayer = player
	return nil
}

func (f *fakeStateClient) GetPlayer(_ context.Context, _ int64) (*statecontract.Player, error) {
	return f.player, nil
}

func (f *fakeStateClient) NextPlayerID(_ context.Context) (int64, error) {
	return f.nextPlayerID, nil
}

var _ statecontract.Client = (*fakeStateClient)(nil)
