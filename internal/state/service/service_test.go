package service

import (
	"context"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestServiceForwardsAccountOperations(t *testing.T) {
	stores := newFakeStores()
	svc := newTestService(stores)
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
	svc := newTestService(stores)
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
	svc := newTestService(stores)

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

func TestServiceForwardsPresenceOperations(t *testing.T) {
	stores := newFakeStores()
	svc := newTestService(stores)
	presence := &statecontract.Presence{
		PlayerID:   7,
		ServerName: "logic-1",
		Status:     "online",
		UpdatedAt:  time.Unix(100, 0),
	}

	if err := svc.SetPresence(context.Background(), presence, time.Minute); err != nil {
		t.Fatalf("SetPresence returned error: %v", err)
	}
	got, err := svc.GetPresence(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetPresence returned error: %v", err)
	}
	if got.ServerName != presence.ServerName {
		t.Fatalf("server name = %q, want %q", got.ServerName, presence.ServerName)
	}
	if err := svc.ClearPresence(context.Background(), 7, "logic-1"); err != nil {
		t.Fatalf("ClearPresence returned error: %v", err)
	}
	if _, err := svc.GetPresence(context.Background(), 7); err == nil {
		t.Fatalf("GetPresence returned nil error, want missing presence error")
	}

	refreshedAt := time.Unix(200, 0)
	if err := svc.RefreshPresence(context.Background(), 8, "logic-2", refreshedAt, 2*time.Minute); err != nil {
		t.Fatalf("RefreshPresence returned error: %v", err)
	}
	if stores.refreshedPlayerID != 8 {
		t.Fatalf("refreshed player id = %d, want 8", stores.refreshedPlayerID)
	}
	if stores.refreshedServerName != "logic-2" {
		t.Fatalf("refreshed server name = %q, want logic-2", stores.refreshedServerName)
	}
	if !stores.refreshedAt.Equal(refreshedAt) {
		t.Fatalf("refreshed at = %v, want %v", stores.refreshedAt, refreshedAt)
	}
	if stores.refreshedTTL != 2*time.Minute {
		t.Fatalf("refreshed ttl = %v, want %v", stores.refreshedTTL, 2*time.Minute)
	}
}

func TestServiceForwardsFriendOperations(t *testing.T) {
	stores := newFakeStores()
	createdAt := time.Unix(300, 0)
	stores.incomingFriendRequests = []*statecontract.FriendRequest{
		{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: createdAt},
	}
	stores.outgoingFriendRequests = []*statecontract.FriendRequest{
		{FromPlayerID: 8, ToPlayerID: 9, CreatedAt: createdAt.Add(time.Minute)},
	}
	stores.friendIDs = []int64{10, 11}
	svc := newTestService(stores)

	if err := svc.SendFriendRequest(context.Background(), 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}
	if stores.sentFriendRequestFrom != 7 {
		t.Fatalf("sent friend request from = %d, want 7", stores.sentFriendRequestFrom)
	}
	if stores.sentFriendRequestTo != 8 {
		t.Fatalf("sent friend request to = %d, want 8", stores.sentFriendRequestTo)
	}

	incoming, err := svc.ListIncomingFriendRequests(context.Background(), 8)
	if err != nil {
		t.Fatalf("ListIncomingFriendRequests returned error: %v", err)
	}
	if stores.listIncomingFriendRequestsPlayerID != 8 {
		t.Fatalf("incoming player id = %d, want 8", stores.listIncomingFriendRequestsPlayerID)
	}
	if len(incoming) != 1 || incoming[0].FromPlayerID != 7 {
		t.Fatalf("incoming requests = %+v, want one request from 7", incoming)
	}

	outgoing, err := svc.ListOutgoingFriendRequests(context.Background(), 8)
	if err != nil {
		t.Fatalf("ListOutgoingFriendRequests returned error: %v", err)
	}
	if stores.listOutgoingFriendRequestsPlayerID != 8 {
		t.Fatalf("outgoing player id = %d, want 8", stores.listOutgoingFriendRequestsPlayerID)
	}
	if len(outgoing) != 1 || outgoing[0].ToPlayerID != 9 {
		t.Fatalf("outgoing requests = %+v, want one request to 9", outgoing)
	}

	if err := svc.AcceptFriendRequest(context.Background(), 7, 8); err != nil {
		t.Fatalf("AcceptFriendRequest returned error: %v", err)
	}
	if stores.acceptedFriendRequestFrom != 7 {
		t.Fatalf("accepted from player = %d, want 7", stores.acceptedFriendRequestFrom)
	}
	if stores.acceptedFriendRequestTo != 8 {
		t.Fatalf("accepted to player = %d, want 8", stores.acceptedFriendRequestTo)
	}

	if err := svc.RejectFriendRequest(context.Background(), 9, 8); err != nil {
		t.Fatalf("RejectFriendRequest returned error: %v", err)
	}
	if stores.rejectedFriendRequestFrom != 9 {
		t.Fatalf("rejected from player = %d, want 9", stores.rejectedFriendRequestFrom)
	}
	if stores.rejectedFriendRequestTo != 8 {
		t.Fatalf("rejected to player = %d, want 8", stores.rejectedFriendRequestTo)
	}

	friendIDs, err := svc.ListFriendIDs(context.Background(), 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	if stores.listFriendIDsPlayerID != 7 {
		t.Fatalf("list friend ids player id = %d, want 7", stores.listFriendIDsPlayerID)
	}
	if len(friendIDs) != 2 || friendIDs[0] != 10 || friendIDs[1] != 11 {
		t.Fatalf("friend ids = %v, want [10 11]", friendIDs)
	}

	if err := svc.DeleteFriend(context.Background(), 7, 10); err != nil {
		t.Fatalf("DeleteFriend returned error: %v", err)
	}
	if stores.deletedFriendPlayerID != 7 {
		t.Fatalf("delete friend player id = %d, want 7", stores.deletedFriendPlayerID)
	}
	if stores.deletedFriendFriendPlayerID != 10 {
		t.Fatalf("delete friend friend player id = %d, want 10", stores.deletedFriendFriendPlayerID)
	}
}

func TestServiceRegisterAccountCreatesAccountPlayerAndSession(t *testing.T) {
	stores := newFakeStores()
	stores.nextPlayerID = 7
	svc := newTestService(stores)
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
	svc := newTestService(stores)

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
	accounts                           map[string]*statecontract.Account
	sessions                           map[string]*statecontract.Session
	players                            map[int64]*statecontract.Player
	presences                          map[int64]*statecontract.Presence
	nextPlayerID                       int64
	refreshedPlayerID                  int64
	refreshedServerName                string
	refreshedAt                        time.Time
	refreshedTTL                       time.Duration
	sentFriendRequestFrom              int64
	sentFriendRequestTo                int64
	listIncomingFriendRequestsPlayerID int64
	listOutgoingFriendRequestsPlayerID int64
	incomingFriendRequests             []*statecontract.FriendRequest
	outgoingFriendRequests             []*statecontract.FriendRequest
	acceptedFriendRequestFrom          int64
	acceptedFriendRequestTo            int64
	rejectedFriendRequestFrom          int64
	rejectedFriendRequestTo            int64
	listFriendIDsPlayerID              int64
	friendIDs                          []int64
	deletedFriendPlayerID              int64
	deletedFriendFriendPlayerID        int64
}

func newFakeStores() *fakeStores {
	return &fakeStores{
		accounts:  make(map[string]*statecontract.Account),
		sessions:  make(map[string]*statecontract.Session),
		players:   make(map[int64]*statecontract.Player),
		presences: make(map[int64]*statecontract.Presence),
	}
}

func newTestService(stores *fakeStores) *Service {
	return NewService(StoreConfig{
		Accounts:      stores,
		Sessions:      stores,
		Players:       stores,
		Presences:     stores,
		Registrations: stores,
		Friends:       stores,
	})
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

func (f *fakeStores) RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	if _, err := f.GetAccount(ctx, input.Username); err == nil {
		return nil, statecontract.ErrAccountExists
	} else if err != statecontract.ErrAccountNotFound {
		return nil, err
	}

	playerID, err := f.NextPlayerID(ctx)
	if err != nil {
		return nil, err
	}
	player := &statecontract.Player{
		ID:       playerID,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}
	account := &statecontract.Account{
		Username:     input.Username,
		PasswordHash: input.PasswordHash,
		PlayerID:     playerID,
	}
	session := &statecontract.Session{
		Token:     input.SessionToken,
		PlayerID:  playerID,
		ExpiresAt: input.SessionExpiresAt,
	}
	if err := f.CreatePlayer(ctx, player); err != nil {
		return nil, err
	}
	if err := f.CreateAccount(ctx, account); err != nil {
		return nil, err
	}
	if err := f.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &statecontract.RegisterAccountResult{
		Account: account,
		Player:  player,
		Session: session,
	}, nil
}

func (f *fakeStores) SetPresence(_ context.Context, presence *statecontract.Presence, _ time.Duration) error {
	cp := *presence
	f.presences[presence.PlayerID] = &cp
	return nil
}

func (f *fakeStores) GetPresence(_ context.Context, playerID int64) (*statecontract.Presence, error) {
	presence, ok := f.presences[playerID]
	if !ok {
		return nil, statecontract.ErrPresenceNotFound
	}
	cp := *presence
	return &cp, nil
}

func (f *fakeStores) ClearPresence(_ context.Context, playerID int64, serverName string) error {
	presence, ok := f.presences[playerID]
	if !ok {
		return nil
	}
	if presence.ServerName == serverName {
		delete(f.presences, playerID)
	}
	return nil
}

func (f *fakeStores) RefreshPresence(_ context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	f.refreshedPlayerID = playerID
	f.refreshedServerName = serverName
	f.refreshedAt = updatedAt
	f.refreshedTTL = ttl
	return nil
}

func (f *fakeStores) SendFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.sentFriendRequestFrom = fromPlayerID
	f.sentFriendRequestTo = toPlayerID
	return nil
}

func (f *fakeStores) ListIncomingFriendRequests(_ context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	f.listIncomingFriendRequestsPlayerID = playerID
	return f.incomingFriendRequests, nil
}

func (f *fakeStores) ListOutgoingFriendRequests(_ context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	f.listOutgoingFriendRequestsPlayerID = playerID
	return f.outgoingFriendRequests, nil
}

func (f *fakeStores) AcceptFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.acceptedFriendRequestFrom = fromPlayerID
	f.acceptedFriendRequestTo = toPlayerID
	return nil
}

func (f *fakeStores) RejectFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.rejectedFriendRequestFrom = fromPlayerID
	f.rejectedFriendRequestTo = toPlayerID
	return nil
}

func (f *fakeStores) ListFriendIDs(_ context.Context, playerID int64) ([]int64, error) {
	f.listFriendIDsPlayerID = playerID
	return f.friendIDs, nil
}

func (f *fakeStores) DeleteFriend(_ context.Context, playerID, friendPlayerID int64) error {
	f.deletedFriendPlayerID = playerID
	f.deletedFriendFriendPlayerID = friendPlayerID
	return nil
}

var _ statecontract.Client = (*Service)(nil)
var _ statecontract.PresenceClient = (*Service)(nil)
var _ statecontract.FriendClient = (*Service)(nil)
