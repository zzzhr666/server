package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	playerpkg "server/internal/logic/player"
	"server/internal/logic/presence"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestHealth(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	wantBody := "ok server_name = logic-test"
	if rec.Body.String() != wantBody {
		t.Fatalf("body = %q, want %q", rec.Body.String(), wantBody)
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

func TestSendFriendRequestHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	friends := newFakeFriendService()
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodPost, "/friends/requests", strings.NewReader(`{"to_player_id":8}`))
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if friends.sentFromPlayerID != 7 || friends.sentToPlayerID != 8 {
		t.Fatalf("send request got from=%d to=%d, want from=7 to=8", friends.sentFromPlayerID, friends.sentToPlayerID)
	}
}

func TestSendFriendRequestHTTPInvalidJSON(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	handler := newTestHandlerWithFriend(auths, newFakeFriendService()).Routes()
	req := httptest.NewRequest(http.MethodPost, "/friends/requests", strings.NewReader(`{"to_player_id":`))
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestSendFriendRequestHTTPConflict(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	friends := newFakeFriendService()
	friends.sendErr = friend.ErrRequestExists
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodPost, "/friends/requests", strings.NewReader(`{"to_player_id":8}`))
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestFriendHTTPMissingToken(t *testing.T) {
	handler := newTestHandlerWithFriend(newFakeAuthService(), newFakeFriendService()).Routes()
	req := httptest.NewRequest(http.MethodGet, "/friends", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestListIncomingFriendRequestsHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(8)
	createdAt := time.Date(2026, 7, 20, 10, 30, 0, 123, time.UTC)
	friends := newFakeFriendService()
	friends.incomingRequests = []*friend.Request{
		{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: createdAt},
	}
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodGet, "/friends/requests/incoming", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if friends.listIncomingPlayerID != 8 {
		t.Fatalf("list incoming player id = %d, want 8", friends.listIncomingPlayerID)
	}
	var resp friendRequestsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Requests) != 1 {
		t.Fatalf("requests count = %d, want 1", len(resp.Requests))
	}
	got := resp.Requests[0]
	if got.FromPlayerID != 7 || got.ToPlayerID != 8 || got.CreatedAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("request = %+v, want from=7 to=8 created_at=%q", got, createdAt.Format(time.RFC3339Nano))
	}
}

func TestListOutgoingFriendRequestsHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	createdAt := time.Date(2026, 7, 20, 10, 31, 0, 0, time.UTC)
	friends := newFakeFriendService()
	friends.outgoingRequests = []*friend.Request{
		{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: createdAt},
	}
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodGet, "/friends/requests/outgoing", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if friends.listOutgoingPlayerID != 7 {
		t.Fatalf("list outgoing player id = %d, want 7", friends.listOutgoingPlayerID)
	}
	var resp friendRequestsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Requests) != 1 || resp.Requests[0].ToPlayerID != 8 {
		t.Fatalf("requests = %+v, want one outgoing request to 8", resp.Requests)
	}
}

func TestAcceptFriendRequestHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(8)
	friends := newFakeFriendService()
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodPost, "/friends/requests/accept", strings.NewReader(`{"from_player_id":7}`))
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if friends.acceptedFromPlayerID != 7 || friends.acceptedToPlayerID != 8 {
		t.Fatalf("accept request got from=%d to=%d, want from=7 to=8", friends.acceptedFromPlayerID, friends.acceptedToPlayerID)
	}
}

func TestRejectFriendRequestHTTPNotFound(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(8)
	friends := newFakeFriendService()
	friends.rejectErr = friend.ErrRequestNotFound
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodPost, "/friends/requests/reject", strings.NewReader(`{"from_player_id":7}`))
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestListFriendsHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	friends := newFakeFriendService()
	friends.friendIDs = []int64{8, 9}
	players := newFakePlayerService()
	players.players[8] = &playerpkg.Player{ID: 8, Nickname: "Bob", Avatar: "bob.png"}
	players.players[9] = &playerpkg.Player{ID: 9, Nickname: "Carol", Avatar: "carol.png"}
	presences := newFakePresenceService()
	presences.presences[8] = &presence.Presence{
		PlayerID:   8,
		ServerName: "logic-other",
		Status:     presence.StatusOnline,
		UpdatedAt:  time.Date(2026, 7, 20, 10, 40, 0, 0, time.UTC),
	}
	handler := newTestHandlerWithAllServices(auths, presences, friends, players).Routes()
	req := httptest.NewRequest(http.MethodGet, "/friends", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if friends.listFriendIDsPlayerID != 7 {
		t.Fatalf("list friends player id = %d, want 7", friends.listFriendIDsPlayerID)
	}
	var resp friendSummariesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Friends) != 2 {
		t.Fatalf("friends count = %d, want 2: %+v", len(resp.Friends), resp.Friends)
	}
	if resp.Friends[0].PlayerID != 8 || resp.Friends[0].Nickname != "Bob" || resp.Friends[0].Avatar != "bob.png" || !resp.Friends[0].Online || resp.Friends[0].Status != presence.StatusOnline {
		t.Fatalf("first friend = %+v, want online Bob", resp.Friends[0])
	}
	if resp.Friends[1].PlayerID != 9 || resp.Friends[1].Nickname != "Carol" || resp.Friends[1].Avatar != "carol.png" || resp.Friends[1].Online || resp.Friends[1].Status != "offline" {
		t.Fatalf("second friend = %+v, want offline Carol", resp.Friends[1])
	}
}

func TestDeleteFriendHTTP(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	friends := newFakeFriendService()
	handler := newTestHandlerWithFriend(auths, friends).Routes()
	req := httptest.NewRequest(http.MethodDelete, "/friends", strings.NewReader(`{"friend_player_id":8}`))
	req.Header.Set("Authorization", "Bearer "+session.Token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if friends.deletedPlayerID != 7 || friends.deletedFriendID != 8 {
		t.Fatalf("delete friend got player=%d friend=%d, want player=7 friend=8", friends.deletedPlayerID, friends.deletedFriendID)
	}
}

func TestPublishFriendPresenceChangedSendsToOnlineFriends(t *testing.T) {
	auths := newFakeAuthService()
	friends := newFakeFriendService()
	friends.friendIDs = []int64{8, 9}
	presences := newFakePresenceService()
	presences.presences[8] = &presence.Presence{
		PlayerID:   8,
		ServerName: "logic-2",
		Status:     presence.StatusOnline,
	}
	realtime := newFakeRealtimeClient()
	handler := NewHandler(HandlerConfig{
		AuthService:     auths,
		ServerName:      "logic-test",
		PresenceService: presences,
		FriendService:   friends,
		PlayerService:   newFakePlayerService(),
		RealtimeClient:  realtime,
	})

	handler.publishFriendPresenceChanged(context.Background(), 7, true, presence.StatusOnline)

	if len(realtime.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(realtime.published))
	}
	got := realtime.published[0]
	if got.serverName != "logic-2" {
		t.Fatalf("published server name = %q, want logic-2", got.serverName)
	}
	if got.event.Type != statecontract.RealtimeEventFriendPresenceChanged || got.event.TargetPlayerID != 8 || got.event.ActorPlayerID != 7 || !got.event.Online {
		t.Fatalf("published event = %+v, want online presence change for friend 8 by player 7", got.event)
	}
}

func TestPublishFriendRemovedSendsOnlyToRemovedPlayer(t *testing.T) {
	auths := newFakeAuthService()
	friends := newFakeFriendService()
	presences := newFakePresenceService()
	presences.presences[8] = &presence.Presence{
		PlayerID:   8,
		ServerName: "logic-2",
		Status:     presence.StatusOnline,
	}
	realtime := newFakeRealtimeClient()
	handler := NewHandler(HandlerConfig{
		AuthService:     auths,
		ServerName:      "logic-test",
		PresenceService: presences,
		FriendService:   friends,
		PlayerService:   newFakePlayerService(),
		RealtimeClient:  realtime,
	})

	handler.publishFriendRemoved(context.Background(), 8, 7)

	if len(realtime.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(realtime.published))
	}
	got := realtime.published[0]
	if got.serverName != "logic-2" {
		t.Fatalf("published server name = %q, want logic-2", got.serverName)
	}
	if got.event.Type != statecontract.RealtimeEventFriendRemoved || got.event.TargetPlayerID != 8 || got.event.ActorPlayerID != 7 {
		t.Fatalf("published event = %+v, want friend_removed target=8 actor=7", got.event)
	}
}

func TestPublishFriendRequestReceivedSendsToRequestTarget(t *testing.T) {
	auths := newFakeAuthService()
	presences := newFakePresenceService()
	presences.presences[8] = &presence.Presence{
		PlayerID:   8,
		ServerName: "logic-2",
		Status:     presence.StatusOnline,
	}
	realtime := newFakeRealtimeClient()
	handler := NewHandler(HandlerConfig{
		AuthService:     auths,
		ServerName:      "logic-test",
		PresenceService: presences,
		FriendService:   newFakeFriendService(),
		PlayerService:   newFakePlayerService(),
		RealtimeClient:  realtime,
	})

	handler.publishFriendRequestReceived(context.Background(), 8, 7)

	if len(realtime.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(realtime.published))
	}
	got := realtime.published[0]
	if got.serverName != "logic-2" {
		t.Fatalf("published server name = %q, want logic-2", got.serverName)
	}
	if got.event.Type != statecontract.RealtimeEventFriendRequestReceived || got.event.TargetPlayerID != 8 || got.event.ActorPlayerID != 7 {
		t.Fatalf("published event = %+v, want friend_request_received target=8 actor=7", got.event)
	}
}

func TestPublishFriendRequestHandledSendsToRequestSender(t *testing.T) {
	auths := newFakeAuthService()
	presences := newFakePresenceService()
	presences.presences[7] = &presence.Presence{
		PlayerID:   7,
		ServerName: "logic-1",
		Status:     presence.StatusOnline,
	}
	realtime := newFakeRealtimeClient()
	handler := NewHandler(HandlerConfig{
		AuthService:     auths,
		ServerName:      "logic-test",
		PresenceService: presences,
		FriendService:   newFakeFriendService(),
		PlayerService:   newFakePlayerService(),
		RealtimeClient:  realtime,
	})

	handler.publishFriendRequestHandled(context.Background(), 7, 8)

	if len(realtime.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(realtime.published))
	}
	got := realtime.published[0]
	if got.serverName != "logic-1" {
		t.Fatalf("published server name = %q, want logic-1", got.serverName)
	}
	if got.event.Type != statecontract.RealtimeEventFriendRequestHandled || got.event.TargetPlayerID != 7 || got.event.ActorPlayerID != 8 {
		t.Fatalf("published event = %+v, want friend_request_handled target=7 actor=8", got.event)
	}
}

func TestReplaceExistingConnectionPublishesConnectionReplaced(t *testing.T) {
	auths := newFakeAuthService()
	presences := newFakePresenceService()
	presences.presences[7] = &presence.Presence{
		PlayerID:   7,
		ServerName: "logic-old",
		Status:     presence.StatusOnline,
	}
	realtime := newFakeRealtimeClient()
	handler := NewHandler(HandlerConfig{
		AuthService:     auths,
		ServerName:      "logic-new",
		PresenceService: presences,
		FriendService:   newFakeFriendService(),
		PlayerService:   newFakePlayerService(),
		RealtimeClient:  realtime,
	})

	handler.replaceExistingConnection(context.Background(), 7)

	if len(realtime.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(realtime.published))
	}
	got := realtime.published[0]
	if got.serverName != "logic-old" {
		t.Fatalf("published server name = %q, want logic-old", got.serverName)
	}
	if got.event.Type != statecontract.RealtimeEventConnectionReplaced || got.event.TargetPlayerID != 7 || got.event.ActorPlayerID != 7 {
		t.Fatalf("published event = %+v, want connection_replaced target=7 actor=7", got.event)
	}
}

func TestToWebSocketEventConnectionReplaced(t *testing.T) {
	msg := toWebSocketEvent(statecontract.RealtimeEvent{
		Type:           statecontract.RealtimeEventConnectionReplaced,
		TargetPlayerID: 7,
	})
	got, ok := msg.(connectionReplacedMessage)
	if !ok {
		t.Fatalf("message type = %T, want connectionReplacedMessage", msg)
	}
	if got.Type != statecontract.RealtimeEventConnectionReplaced {
		t.Fatalf("message type field = %q, want %q", got.Type, statecontract.RealtimeEventConnectionReplaced)
	}
}

func TestWebSocketMissingToken(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestWebSocketInvalidToken(t *testing.T) {
	handler := newTestHandler().Routes()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("token", "missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestWebSocketMarksOnlineAndOffline(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	presences := newFakePresenceService()
	handler := newTestHandlerWithServices(auths, presences).Routes()
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"token": []string{session.Token}},
	})
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer func() {
		_ = conn.CloseNow()
	}()

	onlineCall := waitPresenceCall(t, presences.onlineCalls)
	if onlineCall.PlayerID != 7 {
		t.Fatalf("online player id = %d, want 7", onlineCall.PlayerID)
	}
	if onlineCall.ServerName != "logic-test" {
		t.Fatalf("online server name = %q, want %q", onlineCall.ServerName, "logic-test")
	}

	if err := conn.Close(websocket.StatusNormalClosure, "test done"); err != nil {
		t.Fatalf("close websocket: %v", err)
	}

	offlineCall := waitPresenceCall(t, presences.offlineCalls)
	if offlineCall.PlayerID != 7 {
		t.Fatalf("offline player id = %d, want 7", offlineCall.PlayerID)
	}
	if offlineCall.ServerName != "logic-test" {
		t.Fatalf("offline server name = %q, want %q", offlineCall.ServerName, "logic-test")
	}
}

func TestWebSocketOldConnectionDoesNotMarkOfflineAfterReconnect(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	presences := newFakePresenceService()
	handler := newTestHandlerWithServices(auths, presences).Routes()
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialOptions := &websocket.DialOptions{
		HTTPHeader: http.Header{"token": []string{session.Token}},
	}

	oldConn, _, err := websocket.Dial(ctx, wsURL, dialOptions)
	if err != nil {
		t.Fatalf("old websocket dial: %v", err)
	}
	defer func() {
		_ = oldConn.CloseNow()
	}()
	waitPresenceCall(t, presences.onlineCalls)

	newConn, _, err := websocket.Dial(ctx, wsURL, dialOptions)
	if err != nil {
		t.Fatalf("new websocket dial: %v", err)
	}
	defer func() {
		_ = newConn.CloseNow()
	}()
	waitPresenceCall(t, presences.onlineCalls)

	if err := oldConn.Close(websocket.StatusNormalClosure, "old connection closed"); err != nil {
		t.Fatalf("close old websocket: %v", err)
	}
	assertNoPresenceCall(t, presences.offlineCalls)

	if err := newConn.Close(websocket.StatusNormalClosure, "new connection closed"); err != nil {
		t.Fatalf("close new websocket: %v", err)
	}
	offlineCall := waitPresenceCall(t, presences.offlineCalls)
	if offlineCall.PlayerID != 7 {
		t.Fatalf("offline player id = %d, want 7", offlineCall.PlayerID)
	}
}

func TestWebSocketHeartbeatRefreshesPresenceAndConnection(t *testing.T) {
	auths := newFakeAuthService()
	session := auths.newSession(7)
	presences := newFakePresenceService()
	testHandler := newTestHandlerWithServices(auths, presences)
	server := httptest.NewServer(testHandler.Routes())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"token": []string{session.Token}},
	})
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer func() {
		_ = conn.CloseNow()
	}()
	waitPresenceCall(t, presences.onlineCalls)

	before, ok := testHandler.connections.Get(7)
	if !ok {
		t.Fatalf("connection missing before heartbeat")
	}
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"heartbeat"}`)); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	refreshCall := waitPresenceCall(t, presences.refreshCalls)
	if refreshCall.PlayerID != 7 {
		t.Fatalf("refresh player id = %d, want 7", refreshCall.PlayerID)
	}
	if refreshCall.ServerName != "logic-test" {
		t.Fatalf("refresh server name = %q, want logic-test", refreshCall.ServerName)
	}
	after, ok := testHandler.connections.Get(7)
	if !ok {
		t.Fatalf("connection missing after heartbeat")
	}
	if !after.lastHeartbeatAt.After(before.lastHeartbeatAt) {
		t.Fatalf("last heartbeat at = %v, want after %v", after.lastHeartbeatAt, before.lastHeartbeatAt)
	}
}

func newTestHandler() *Handler {
	return newTestHandlerWithAuth(newFakeAuthService())
}

func newTestHandlerWithAuth(auths *fakeAuthService) *Handler {
	return newTestHandlerWithServices(auths, newFakePresenceService())
}

func newTestHandlerWithServices(auths *fakeAuthService, presences presence.Service) *Handler {
	return newTestHandlerWithAllServices(auths, presences, newFakeFriendService(), newFakePlayerService())
}

func newTestHandlerWithFriend(auths *fakeAuthService, friends friend.Service) *Handler {
	return newTestHandlerWithAllServices(auths, newFakePresenceService(), friends, newFakePlayerService())
}

func newTestHandlerWithAllServices(auths *fakeAuthService, presences presence.Service, friends friend.Service, players playerpkg.Service) *Handler {
	return NewHandler(HandlerConfig{
		AuthService:     auths,
		PresenceService: presences,
		FriendService:   friends,
		PlayerService:   players,
		ServerName:      "logic-test",
	})
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

type presenceCall struct {
	PlayerID   int64
	ServerName string
}

type fakePresenceService struct {
	markOnlineErr  error
	markOfflineErr error
	onlineCalls    chan presenceCall
	offlineCalls   chan presenceCall
	refreshCalls   chan presenceCall
	presences      map[int64]*presence.Presence
}

func newFakePresenceService() *fakePresenceService {
	return &fakePresenceService{
		onlineCalls:  make(chan presenceCall, 4),
		offlineCalls: make(chan presenceCall, 4),
		refreshCalls: make(chan presenceCall, 4),
		presences:    make(map[int64]*presence.Presence),
	}
}

func waitPresenceCall(t *testing.T, calls <-chan presenceCall) presenceCall {
	t.Helper()

	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for presence call")
		return presenceCall{}
	}
}

func assertNoPresenceCall(t *testing.T, calls <-chan presenceCall) {
	t.Helper()

	select {
	case call := <-calls:
		t.Fatalf("unexpected presence call: %+v", call)
	case <-time.After(100 * time.Millisecond):
	}
}

func (f *fakePresenceService) MarkOnline(ctx context.Context, playerID int64, serverName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.recordCall(f.onlineCalls, playerID, serverName)
	return f.markOnlineErr
}

func (f *fakePresenceService) Get(ctx context.Context, playerID int64) (*presence.Presence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, ok := f.presences[playerID]
	if !ok {
		return nil, presence.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakePresenceService) MarkOffline(ctx context.Context, playerID int64, serverName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.recordCall(f.offlineCalls, playerID, serverName)
	return f.markOfflineErr
}

func (f *fakePresenceService) Refresh(ctx context.Context, playerID int64, serverName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.recordCall(f.refreshCalls, playerID, serverName)
	return nil
}

func (f *fakePresenceService) recordCall(calls chan presenceCall, playerID int64, serverName string) {
	if calls == nil {
		return
	}
	select {
	case calls <- presenceCall{PlayerID: playerID, ServerName: serverName}:
	default:
	}
}

var _ presence.Service = (*fakePresenceService)(nil)

type fakePlayerService struct {
	players map[int64]*playerpkg.Player
	err     error
}

func newFakePlayerService() *fakePlayerService {
	return &fakePlayerService{
		players: make(map[int64]*playerpkg.Player),
	}
}

func (f *fakePlayerService) Create(ctx context.Context, input playerpkg.CreateInput) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	player := &playerpkg.Player{
		ID:       int64(len(f.players) + 1),
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}
	f.players[player.ID] = player
	return clonePlayer(player), nil
}

func (f *fakePlayerService) Get(ctx context.Context, id int64) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	player, ok := f.players[id]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	return clonePlayer(player), nil
}

var _ playerpkg.Service = (*fakePlayerService)(nil)

type fakeFriendService struct {
	sendErr               error
	listIncomingErr       error
	listOutgoingErr       error
	acceptErr             error
	rejectErr             error
	listFriendIDsErr      error
	deleteErr             error
	sentFromPlayerID      int64
	sentToPlayerID        int64
	listIncomingPlayerID  int64
	listOutgoingPlayerID  int64
	incomingRequests      []*friend.Request
	outgoingRequests      []*friend.Request
	acceptedFromPlayerID  int64
	acceptedToPlayerID    int64
	rejectedFromPlayerID  int64
	rejectedToPlayerID    int64
	listFriendIDsPlayerID int64
	friendIDs             []int64
	deletedPlayerID       int64
	deletedFriendID       int64
}

func newFakeFriendService() *fakeFriendService {
	return &fakeFriendService{}
}

func (f *fakeFriendService) SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.sentFromPlayerID = fromPlayerID
	f.sentToPlayerID = toPlayerID
	return f.sendErr
}

func (f *fakeFriendService) ListIncomingRequests(ctx context.Context, playerID int64) ([]*friend.Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.listIncomingPlayerID = playerID
	return f.incomingRequests, f.listIncomingErr
}

func (f *fakeFriendService) ListOutgoingRequests(ctx context.Context, playerID int64) ([]*friend.Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.listOutgoingPlayerID = playerID
	return f.outgoingRequests, f.listOutgoingErr
}

func (f *fakeFriendService) AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.acceptedFromPlayerID = fromPlayerID
	f.acceptedToPlayerID = toPlayerID
	return f.acceptErr
}

func (f *fakeFriendService) RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.rejectedFromPlayerID = fromPlayerID
	f.rejectedToPlayerID = toPlayerID
	return f.rejectErr
}

func (f *fakeFriendService) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.listFriendIDsPlayerID = playerID
	return f.friendIDs, f.listFriendIDsErr
}

func (f *fakeFriendService) DeleteFriend(ctx context.Context, playerID, friendID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.deletedPlayerID = playerID
	f.deletedFriendID = friendID
	return f.deleteErr
}

var _ friend.Service = (*fakeFriendService)(nil)

type publishedRealtimeEvent struct {
	serverName string
	event      statecontract.RealtimeEvent
}

type fakeRealtimeClient struct {
	published []publishedRealtimeEvent
}

func newFakeRealtimeClient() *fakeRealtimeClient {
	return &fakeRealtimeClient{}
}

func (f *fakeRealtimeClient) PublishRealtimeToServer(ctx context.Context, serverName string, event *statecontract.RealtimeEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event == nil {
		return nil
	}
	f.published = append(f.published, publishedRealtimeEvent{
		serverName: serverName,
		event:      *event,
	})
	return nil
}

func (f *fakeRealtimeClient) SubscribeRealtime(ctx context.Context, _ string) (<-chan *statecontract.RealtimeEvent, error) {
	events := make(chan *statecontract.RealtimeEvent)
	go func() {
		defer close(events)
		<-ctx.Done()
	}()
	return events, nil
}

var _ statecontract.RealtimeClient = (*fakeRealtimeClient)(nil)
