package realtime

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"server/internal/contract/realtimepb"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/match"
	"server/internal/logic/player"
	"server/internal/logic/presence"
	"server/internal/rcenter"

	"google.golang.org/protobuf/proto"
)

func TestHandlerServeSessionAuthenticatesRefreshesAndCleansUp(t *testing.T) {
	authService := &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}}
	presenceService := &fakeHandlerPresence{}
	handler := NewHandler(HandlerConfig{
		AuthService:     authService,
		PresenceService: presenceService,
		ServerName:      "logic-test",
	})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(1, "session-token"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 1 || response.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authentication response = %v, want player 7 for request 1", response)
	}
	if got := presenceService.onlineCallCount(); got != 1 {
		t.Fatalf("MarkOnline calls = %d, want 1", got)
	}
	if _, ok := handler.connManager.Get(7); !ok {
		t.Fatal("authenticated session is not registered")
	}

	writeClientEnvelope(t, clientConn, heartbeatEnvelope(2))
	response = readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 2", response)
	}
	if got := presenceService.refreshCallCount(); got != 1 {
		t.Fatalf("Refresh calls = %d, want 1", got)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
	if got := presenceService.offlineCallCount(); got != 1 {
		t.Fatalf("MarkOffline calls = %d, want 1", got)
	}
	if _, ok := handler.connManager.Get(7); ok {
		t.Fatal("session remains registered after disconnect")
	}
}

func TestHandlerServeSessionTimesOutAuthenticatedIdleConnection(t *testing.T) {
	authService := &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}}
	presenceService := &fakeHandlerPresence{}
	handler := NewHandler(HandlerConfig{
		AuthService:     authService,
		PresenceService: presenceService,
		ServerName:      "logic-test",
		IdleTimeout:     20 * time.Millisecond,
	})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	waitForHandler(t, done)

	if _, ok := handler.connManager.Get(7); ok {
		t.Fatal("idle connection remains registered after timeout")
	}
	if got := presenceService.offlineCallCount(); got != 1 {
		t.Fatalf("MarkOffline calls = %d, want 1 after idle timeout", got)
	}
}

func TestHandlerServeSessionRefreshesIdleDeadlineAfterBusinessPacket(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	conn := &readDeadlineControlConn{Conn: serverConn}
	handler := NewHandler(HandlerConfig{
		AuthService:     &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		PresenceService: &fakeHandlerPresence{},
		PlayerService:   &handlerMethodPlayerService{result: &player.Player{ID: 7, Nickname: "player-7"}},
		ServerName:      "logic-test",
		IdleTimeout:     time.Second,
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.serveSession(context.Background(), newSession(conn))
	}()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, &realtimepb.ClientEnvelope{
		RequestId: 2,
		Payload: &realtimepb.ClientEnvelope_PlayerGet{
			PlayerGet: &realtimepb.PlayerGetRequest{},
		},
	})
	response := readServerEnvelopeWithDeadline(t, clientConn)
	if response.GetRequestId() != 2 || response.GetPlayer().GetPlayer().GetId() != 7 {
		t.Fatalf("player response = %v, want player 7 for request 2", response)
	}

	deadlines := waitForReadDeadlines(t, conn, 4)
	if deadlines[0].IsZero() || !deadlines[1].IsZero() || deadlines[2].IsZero() || deadlines[3].IsZero() {
		t.Fatalf("read deadlines = %v, want authentication set/clear followed by two idle deadlines", deadlines)
	}
	if !deadlines[3].After(deadlines[2]) {
		t.Fatalf("refreshed idle deadline = %v, want after previous deadline %v", deadlines[3], deadlines[2])
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionRejectsNonAuthenticationFirstMessage(t *testing.T) {
	authService := &fakeHandlerAuth{}
	presenceService := &fakeHandlerPresence{}
	handler := NewHandler(HandlerConfig{AuthService: authService, PresenceService: presenceService, ServerName: "logic-test"})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, heartbeatEnvelope(3))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 3 || response.GetError().GetCode() != realtimepb.ErrorCode_INVALID_ARGUMENT {
		t.Fatalf("error response = %v, want invalid argument for request 3", response)
	}
	waitForHandler(t, done)
	if got := authService.getSessionCallCount(); got != 0 {
		t.Fatalf("GetSession calls = %d, want 0", got)
	}
	if got := presenceService.onlineCallCount(); got != 0 {
		t.Fatalf("MarkOnline calls = %d, want 0", got)
	}
}

func TestHandlerServeSessionTimesOutBeforeAuthentication(t *testing.T) {
	authService := &fakeHandlerAuth{}
	presenceService := &fakeHandlerPresence{}
	handler := NewHandler(HandlerConfig{
		AuthService:           authService,
		PresenceService:       presenceService,
		ServerName:            "logic-test",
		AuthenticationTimeout: 20 * time.Millisecond,
	})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()
	defer clientConn.Close()

	waitForHandler(t, done)
	if got := authService.getSessionCallCount(); got != 0 {
		t.Fatalf("GetSession calls = %d, want 0 after authentication timeout", got)
	}
	if got := presenceService.onlineCallCount(); got != 0 {
		t.Fatalf("MarkOnline calls = %d, want 0 after authentication timeout", got)
	}
}

func TestHandlerAuthenticateAppliesAndClearsReadDeadline(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	conn := &readDeadlineControlConn{Conn: serverConn}
	handler := NewHandler(HandlerConfig{
		AuthService:           &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		AuthenticationTimeout: time.Second,
	})

	result := startHandlerAuthentication(handler, newSession(conn))
	writeClientEnvelope(t, clientConn, authenticateEnvelope(9, "session-token"))
	got := waitForHandlerAuthentication(t, result)

	if !got.ok || got.authSession == nil || got.authSession.PlayerID != 7 || got.requestID != 9 {
		t.Fatalf("handleAuthenticate() = (%v, %d, %t), want player 7, request 9, true", got.authSession, got.requestID, got.ok)
	}
	deadlines := conn.readDeadlines()
	if len(deadlines) != 2 {
		t.Fatalf("SetReadDeadline calls = %d, want 2", len(deadlines))
	}
	if deadlines[0].IsZero() {
		t.Fatal("authentication read deadline = zero, want bounded deadline")
	}
	if !deadlines[1].IsZero() {
		t.Fatalf("cleared authentication read deadline = %v, want zero", deadlines[1])
	}
}

func TestHandlerAuthenticateStopsWhenSettingReadDeadlineFails(t *testing.T) {
	wantErr := errors.New("set read deadline failed")
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	authService := &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}}
	handler := NewHandler(HandlerConfig{AuthService: authService})

	authSession, requestID, ok := handler.handleAuthenticate(context.Background(), newSession(&readDeadlineControlConn{
		Conn:   serverConn,
		setErr: wantErr,
	}))

	if ok || authSession != nil || requestID != 0 {
		t.Fatalf("handleAuthenticate() = (%v, %d, %t), want nil, 0, false", authSession, requestID, ok)
	}
	if got := authService.getSessionCallCount(); got != 0 {
		t.Fatalf("GetSession calls = %d, want 0 after setting read deadline fails", got)
	}
}

func TestHandlerAuthenticateStopsWhenClearingReadDeadlineFails(t *testing.T) {
	wantErr := errors.New("clear read deadline failed")
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	conn := &readDeadlineControlConn{Conn: serverConn, clearErr: wantErr}
	authService := &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}}
	handler := NewHandler(HandlerConfig{AuthService: authService})

	result := startHandlerAuthentication(handler, newSession(conn))
	writeClientEnvelope(t, clientConn, authenticateEnvelope(9, "session-token"))
	got := waitForHandlerAuthentication(t, result)

	if got.ok || got.authSession != nil || got.requestID != 0 {
		t.Fatalf("handleAuthenticate() = (%v, %d, %t), want nil, 0, false", got.authSession, got.requestID, got.ok)
	}
	if calls := authService.getSessionCallCount(); calls != 0 {
		t.Fatalf("GetSession calls = %d, want 0 after clearing read deadline fails", calls)
	}
	deadlines := conn.readDeadlines()
	if len(deadlines) != 2 || deadlines[0].IsZero() || !deadlines[1].IsZero() {
		t.Fatalf("read deadlines = %v, want bounded deadline followed by zero", deadlines)
	}
}

func TestHandlerServeSessionRejectsInvalidSession(t *testing.T) {
	authService := &fakeHandlerAuth{getSessionErr: auth.ErrSessionNotFound}
	presenceService := &fakeHandlerPresence{}
	handler := NewHandler(HandlerConfig{AuthService: authService, PresenceService: presenceService, ServerName: "logic-test"})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(4, "missing"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 4 || response.GetError().GetCode() != realtimepb.ErrorCode_UNAUTHENTICATED {
		t.Fatalf("error response = %v, want unauthenticated for request 4", response)
	}
	waitForHandler(t, done)
	if got := presenceService.onlineCallCount(); got != 0 {
		t.Fatalf("MarkOnline calls = %d, want 0", got)
	}
}

func TestHandlerServeSessionCleansUpWhenMarkOnlineFails(t *testing.T) {
	authService := &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}}
	presenceService := &fakeHandlerPresence{markOnlineErr: errors.New("state unavailable")}
	handler := NewHandler(HandlerConfig{AuthService: authService, PresenceService: presenceService, ServerName: "logic-test"})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(5, "session-token"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 5 || response.GetError().GetCode() != realtimepb.ErrorCode_INTERNAL {
		t.Fatalf("error response = %v, want internal error for request 5", response)
	}
	waitForHandler(t, done)
	if _, ok := handler.connManager.Get(7); ok {
		t.Fatal("session was registered despite MarkOnline failure")
	}
	if got := presenceService.offlineCallCount(); got != 0 {
		t.Fatalf("MarkOffline calls = %d, want 0", got)
	}
}

func TestHandlerServeSessionReplacesConnectionOnExistingLogicServer(t *testing.T) {
	presenceService := &fakeHandlerPresence{getResult: &presence.Presence{PlayerID: 7, ServerName: "logic-old"}}
	realtimeClient := &fakeHandlerRealtimeClient{}
	handler := NewHandler(HandlerConfig{
		AuthService:     &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		PresenceService: presenceService,
		ServerName:      "logic-new",
		RealtimeClient:  realtimeClient,
	})
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(1, "session-token"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authentication response = %v, want player 7", response)
	}
	published, ok := realtimeClient.publishedEvent()
	if !ok {
		t.Fatal("connection replacement event was not published")
	}
	if published.delivery.Route.Type != statecontract.RealtimeRouteServer || published.delivery.Route.ServerName != "logic-old" || published.delivery.Event.Type != statecontract.RealtimeEventConnectionReplaced || published.delivery.Event.TargetPlayerID != 7 || published.delivery.Event.ActorPlayerID != 7 {
		t.Fatalf("published event = %+v, want replacement for player 7 on logic-old", published)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerDoesNotPublishConnectionReplacementToCurrentLogicServer(t *testing.T) {
	presenceService := &fakeHandlerPresence{getResult: &presence.Presence{PlayerID: 7, ServerName: "logic-test"}}
	realtimeClient := &fakeHandlerRealtimeClient{}
	handler := NewHandler(HandlerConfig{
		PresenceService: presenceService,
		ServerName:      "logic-test",
		RealtimeClient:  realtimeClient,
	})

	handler.replaceExistingConnection(context.Background(), 7)

	if published, ok := realtimeClient.publishedEvent(); ok {
		t.Fatalf("published event = %+v, want no cross-server replacement event for current logic server", published)
	}
}

func TestHandlerServeSessionReplacesLocalConnectionWithoutClosingNewConnection(t *testing.T) {
	presenceService := &fakeHandlerPresence{getResult: &presence.Presence{PlayerID: 7, ServerName: "logic-test"}}
	realtimeClient := &fakeHandlerRealtimeClient{}
	handler := NewHandler(HandlerConfig{
		AuthService:     &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		PresenceService: presenceService,
		ServerName:      "logic-test",
		RealtimeClient:  realtimeClient,
	})

	oldServerConn, oldClientConn, oldDone := startHandlerSession(t, handler)
	defer oldServerConn.Close()
	defer oldClientConn.Close()
	authenticateSession(t, oldClientConn)

	newServerConn, newClientConn, newDone := startHandlerSession(t, handler)
	defer newServerConn.Close()
	defer newClientConn.Close()
	writeClientEnvelope(t, newClientConn, authenticateEnvelope(2, "session-token"))

	replaced := readServerEnvelopeWithDeadline(t, oldClientConn)
	if replaced.GetConnectionReplaced() == nil {
		t.Fatalf("old connection response = %v, want connection replaced", replaced)
	}
	waitForHandler(t, oldDone)

	authenticated := readServerEnvelopeWithDeadline(t, newClientConn)
	if authenticated.GetRequestId() != 2 || authenticated.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("new connection response = %v, want authenticated player 7 for request 2", authenticated)
	}
	writeClientEnvelope(t, newClientConn, heartbeatEnvelope(3))
	heartbeat := readServerEnvelopeWithDeadline(t, newClientConn)
	if heartbeat.GetRequestId() != 3 || heartbeat.GetHeartbeatAck() == nil {
		t.Fatalf("new connection heartbeat response = %v, want heartbeat ack for request 3", heartbeat)
	}
	if published, ok := realtimeClient.publishedEvent(); ok {
		t.Fatalf("published event = %+v, want local replacement without realtime publish", published)
	}

	if err := newClientConn.Close(); err != nil {
		t.Fatalf("close new client connection: %v", err)
	}
	waitForHandler(t, newDone)
	if got := presenceService.offlineCallCount(); got != 1 {
		t.Fatalf("MarkOffline calls = %d, want only the current connection to mark offline", got)
	}
}

func TestHandlerServeSessionContinuesWhenConnectionReplacementPublishFails(t *testing.T) {
	presenceService := &fakeHandlerPresence{getResult: &presence.Presence{PlayerID: 7, ServerName: "logic-old"}}
	handler := NewHandler(HandlerConfig{
		AuthService:     &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		PresenceService: presenceService,
		ServerName:      "logic-new",
		RealtimeClient:  &fakeHandlerRealtimeClient{publishErr: errors.New("state unavailable")},
	})
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(1, "session-token"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authentication response = %v, want player 7", response)
	}
	if got := presenceService.onlineCallCount(); got != 1 {
		t.Fatalf("MarkOnline calls = %d, want 1", got)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerPublishesFriendPresenceToFriendServer(t *testing.T) {
	presenceService := &fakeHandlerPresence{getResult: &presence.Presence{PlayerID: 8, ServerName: "logic-friend"}}
	realtimeClient := &fakeHandlerRealtimeClient{}
	handler := NewHandler(HandlerConfig{
		PresenceService: presenceService,
		FriendService:   &fakeHandlerFriend{friendIDs: []int64{8}},
		RealtimeClient:  realtimeClient,
	})

	handler.publishFriendPresenceChanged(context.Background(), 7, true, presence.StatusOnline)
	published, ok := realtimeClient.publishedEvent()
	if !ok || presenceService.getPlayerID != 8 || published.delivery.Route.Type != statecontract.RealtimeRouteServer || published.delivery.Route.ServerName != "logic-friend" || published.delivery.Event.TargetPlayerID != 8 || published.delivery.Event.ActorPlayerID != 7 || !published.delivery.Event.Online {
		t.Fatalf("published friend presence = %+v, want online event for friend 8 on logic-friend", published)
	}
}

func TestHandlerServeSessionStartsMatch(t *testing.T) {
	matchService := &fakeHandlerMatch{startResult: &rcenter.MatchResult{Status: rcenter.MatchStatusWaiting}}
	handler := newHandlerWithMatch(matchService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "axe", true))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetMatchResult().GetStatus() != string(rcenter.MatchStatusWaiting) {
		t.Fatalf("match response = %v, want waiting result for request 2", response)
	}
	if matchService.startPlayerID != 7 || matchService.startWeapon != "axe" || !matchService.startSolo {
		t.Fatalf("Start(playerID, weapon, solo) = (%d, %q, %t), want (7, %q, true)", matchService.startPlayerID, matchService.startWeapon, matchService.startSolo, "axe")
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionPushesMatchedResultToLocalPlayer(t *testing.T) {
	matchResult := &rcenter.MatchResult{
		Status:         rcenter.MatchStatusMatched,
		RoomName:       "room-7-8",
		Token:          "match-token",
		BattleNodeName: "battle-1",
		BattleUDPAddr:  "127.0.0.1:9001",
		PlayerIDs:      []int64{7, 8},
	}
	handler := newHandlerWithMatch(&fakeHandlerMatch{startResult: matchResult})
	otherServerConn, otherClientConn := net.Pipe()
	defer otherServerConn.Close()
	defer otherClientConn.Close()
	handler.connManager.Add(8, newSession(otherServerConn))

	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "axe", false))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetMatchResult().GetRoomName() != "room-7-8" {
		t.Fatalf("match response = %v, want current player's match result", response)
	}

	pushed := readServerEnvelopeWithDeadline(t, otherClientConn)
	if pushed.GetRequestId() != proactivePushRequestID || pushed.GetMatchResult().GetRoomName() != "room-7-8" || pushed.GetMatchResult().GetToken() != "match-token" || pushed.GetMatchResult().GetBattleNodeName() != "battle-1" || pushed.GetMatchResult().GetBattleUdpAddr() != "127.0.0.1:9001" {
		t.Fatalf("pushed match result = %v, want complete proactive result for player 8", pushed)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerPushesMatchedResultToRemotePlayer(t *testing.T) {
	presenceService := &fakeHandlerPresence{getResult: &presence.Presence{PlayerID: 8, ServerName: "logic-remote"}}
	realtimeClient := &fakeHandlerRealtimeClient{}
	handler := newHandlerWithMatchAndPresence(nil, presenceService)
	handler.realtimeClient = realtimeClient

	handler.pushMatchResultToPlayers(context.Background(), 7, &rcenter.MatchResult{
		Status:         rcenter.MatchStatusMatched,
		RoomName:       "room-7-8",
		Token:          "match-token",
		BattleNodeName: "battle-1",
		BattleUDPAddr:  "127.0.0.1:9001",
		PlayerIDs:      []int64{7, 8},
	})

	published, ok := realtimeClient.publishedEvent()
	if !ok {
		t.Fatal("remote match result was not published")
	}
	if presenceService.getPlayerID != 8 || published.delivery.Route.Type != statecontract.RealtimeRouteServer || published.delivery.Route.ServerName != "logic-remote" || published.delivery.Event.Type != statecontract.RealtimeEventMatchResult || published.delivery.Event.TargetPlayerID != 8 || published.delivery.Event.ActorPlayerID != 7 || published.delivery.Event.MatchStatus != string(rcenter.MatchStatusMatched) || published.delivery.Event.RoomName != "room-7-8" || published.delivery.Event.MatchToken != "match-token" || published.delivery.Event.BattleNodeName != "battle-1" || published.delivery.Event.BattleUDPAddr != "127.0.0.1:9001" || !samePlayerIDs(published.delivery.Event.MatchPlayerIDs, []int64{7, 8}) {
		t.Fatalf("published match result = %+v, want complete event for player 8 on logic-remote", published)
	}
}

func TestHandlerDoesNotPushWaitingMatchResult(t *testing.T) {
	presenceService := &fakeHandlerPresence{}
	realtimeClient := &fakeHandlerRealtimeClient{}
	handler := newHandlerWithMatchAndPresence(nil, presenceService)
	handler.realtimeClient = realtimeClient

	handler.pushMatchResultToPlayers(context.Background(), 7, &rcenter.MatchResult{
		Status:    rcenter.MatchStatusWaiting,
		PlayerIDs: []int64{7, 8},
	})

	if _, ok := realtimeClient.publishedEvent(); ok {
		t.Fatal("waiting match result was published")
	}
	if presenceService.getPlayerID != 0 {
		t.Fatalf("presence Get player ID = %d, want no lookup", presenceService.getPlayerID)
	}
}

func TestHandlerServeSessionReturnsErrorWhenMatchStartFails(t *testing.T) {
	presenceService := &fakeHandlerPresence{}
	handler := newHandlerWithMatchAndPresence(&fakeHandlerMatch{startErr: errors.New("queue unavailable")}, presenceService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "axe", false))
	response := readServerEnvelopeWithDeadline(t, clientConn)
	if response.GetRequestId() != 2 || response.GetError().GetCode() != realtimepb.ErrorCode_INTERNAL {
		t.Fatalf("match error response = %v, want internal error for request 2", response)
	}
	writeClientEnvelope(t, clientConn, heartbeatEnvelope(3))
	response = readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 3 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 3", response)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionRejectsInvalidMatchWeapon(t *testing.T) {
	presenceService := &fakeHandlerPresence{}
	handler := newHandlerWithMatchAndPresence(&fakeHandlerMatch{}, presenceService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "wand", false))
	response := readServerEnvelopeWithDeadline(t, clientConn)
	if response.GetRequestId() != 2 || response.GetError().GetCode() != realtimepb.ErrorCode_INVALID_ARGUMENT {
		t.Fatalf("match error response = %v, want invalid argument for request 2", response)
	}
	writeClientEnvelope(t, clientConn, heartbeatEnvelope(3))
	response = readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 3 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 3", response)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionCancelsMatch(t *testing.T) {
	matchService := &fakeHandlerMatch{}
	handler := newHandlerWithMatch(matchService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchCancelEnvelope(2))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetMatchCanceled() == nil {
		t.Fatalf("cancel response = %v, want match canceled for request 2", response)
	}
	if matchService.cancelPlayerID != 7 {
		t.Fatalf("Cancel(playerID) = %d, want 7", matchService.cancelPlayerID)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionReturnsErrorWhenMatchCancelFails(t *testing.T) {
	handler := newHandlerWithMatch(&fakeHandlerMatch{cancelErr: rcenter.ErrPlayerNotWaiting})
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchCancelEnvelope(2))
	response := readServerEnvelopeWithDeadline(t, clientConn)
	if response.GetRequestId() != 2 || response.GetError().GetCode() != realtimepb.ErrorCode_CONFLICT {
		t.Fatalf("cancel error response = %v, want conflict for request 2", response)
	}
	writeClientEnvelope(t, clientConn, heartbeatEnvelope(3))
	response = readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 3 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 3", response)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionResumesMatch(t *testing.T) {
	resumeResult := &rcenter.MatchResult{
		Status:         rcenter.MatchStatusMatched,
		RoomName:       "room-7-8",
		Token:          "room-token",
		BattleNodeName: "battle-1",
		BattleUDPAddr:  "127.0.0.1:7001",
	}
	resumeCalls := 0
	matchService := &fakeHandlerMatch{resumeFunc: func(context.Context, int64) (*rcenter.MatchResult, error) {
		resumeCalls++
		if resumeCalls == 1 {
			return nil, rcenter.ErrActiveMatchNotFound
		}
		return resumeResult, nil
	}}
	handler := newHandlerWithMatch(matchService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchResumeEnvelope(2))
	response := readServerEnvelopeOrFail(t, clientConn)
	matchResult := response.GetMatchResult()
	if response.GetRequestId() != 2 || matchResult == nil {
		t.Fatalf("resume response = %v, want match result for request 2", response)
	}
	if matchResult.GetRoomName() != "room-7-8" || matchResult.GetToken() != "room-token" || matchResult.GetBattleUdpAddr() != "127.0.0.1:7001" {
		t.Fatalf("resume result = %v, want active match allocation", matchResult)
	}
	if matchService.resumePlayerID != 7 {
		t.Fatalf("Resume(playerID) = %d, want 7", matchService.resumePlayerID)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionReturnsNotFoundWhenMatchResumeFails(t *testing.T) {
	handler := newHandlerWithMatch(&fakeHandlerMatch{resumeErr: rcenter.ErrActiveMatchNotFound})
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchResumeEnvelope(2))
	response := readServerEnvelopeWithDeadline(t, clientConn)
	if response.GetRequestId() != 2 || response.GetError().GetCode() != realtimepb.ErrorCode_NOT_FOUND {
		t.Fatalf("resume error response = %v, want not found for request 2", response)
	}
	writeClientEnvelope(t, clientConn, heartbeatEnvelope(3))
	response = readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 3 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 3", response)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionAutomaticallyResumesMatch(t *testing.T) {
	matchService := &fakeHandlerMatch{resumeResult: &rcenter.MatchResult{
		Status:         rcenter.MatchStatusMatched,
		RoomName:       "room-7-8",
		Token:          "room-token",
		BattleNodeName: "battle-1",
		BattleUDPAddr:  "127.0.0.1:7001",
	}}
	handler := newHandlerWithMatch(matchService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(1, "session-token"))
	authResponse := readServerEnvelopeOrFail(t, clientConn)
	if authResponse.GetRequestId() != 1 || authResponse.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authentication response = %v, want player 7 for request 1", authResponse)
	}
	resumeResponse := readServerEnvelopeOrFail(t, clientConn)
	if resumeResponse.GetRequestId() != 0 || resumeResponse.GetMatchResult().GetRoomName() != "room-7-8" {
		t.Fatalf("automatic resume response = %v, want pushed active match", resumeResponse)
	}
	if matchService.resumePlayerID != 7 {
		t.Fatalf("Resume(playerID) = %d, want 7", matchService.resumePlayerID)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionContinuesWithoutAutomaticMatch(t *testing.T) {
	handler := newHandlerWithMatch(&fakeHandlerMatch{resumeErr: rcenter.ErrActiveMatchNotFound})
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, heartbeatEnvelope(2))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 2", response)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func TestHandlerServeSessionContinuesWhenAutomaticResumeFails(t *testing.T) {
	handler := newHandlerWithMatch(&fakeHandlerMatch{resumeErr: errors.New("rcenter unavailable")})
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, heartbeatEnvelope(2))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack for request 2", response)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
}

func newHandlerWithMatch(matchService match.Service) *Handler {
	return newHandlerWithMatchAndPresence(matchService, &fakeHandlerPresence{})
}

func newHandlerWithMatchAndPresence(matchService match.Service, presenceService presence.Service) *Handler {
	return &Handler{
		auth:        &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		presence:    presenceService,
		match:       matchService,
		serverName:  "logic-test",
		connManager: newConnectionManager(),
	}
}

func startHandlerSession(t *testing.T, handler *Handler) (net.Conn, net.Conn, <-chan struct{}) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.serveSession(context.Background(), newSession(serverConn))
	}()
	return serverConn, clientConn, done
}

func waitForHandler(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not stop")
	}
}

type handlerAuthenticationResult struct {
	authSession *auth.Session
	requestID   uint64
	ok          bool
}

func startHandlerAuthentication(handler *Handler, session *session) <-chan handlerAuthenticationResult {
	result := make(chan handlerAuthenticationResult, 1)
	go func() {
		authSession, requestID, ok := handler.handleAuthenticate(context.Background(), session)
		result <- handlerAuthenticationResult{authSession: authSession, requestID: requestID, ok: ok}
	}()
	return result
}

func waitForHandlerAuthentication(t *testing.T, result <-chan handlerAuthenticationResult) handlerAuthenticationResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(time.Second):
		t.Fatal("handleAuthenticate did not return")
		return handlerAuthenticationResult{}
	}
}

func authenticateEnvelope(requestID uint64, token string) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_Authenticate{
			Authenticate: &realtimepb.AuthenticateRequest{Token: token},
		},
	}
}

func heartbeatEnvelope(requestID uint64) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_Heartbeat{
			Heartbeat: &realtimepb.HeartbeatRequest{},
		},
	}
}

func matchStartEnvelope(requestID uint64, weapon string, solo bool) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_MatchStart{
			MatchStart: &realtimepb.MatchStartRequest{Weapon: weapon, Solo: solo},
		},
	}
}

func matchCancelEnvelope(requestID uint64) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_MatchCancel{
			MatchCancel: &realtimepb.MatchCancelRequest{},
		},
	}
}

func matchResumeEnvelope(requestID uint64) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_MatchResume{
			MatchResume: &realtimepb.MatchResumeRequest{},
		},
	}
}

func authenticateSession(t *testing.T, conn net.Conn) {
	t.Helper()
	writeClientEnvelope(t, conn, authenticateEnvelope(1, "session-token"))
	response := readServerEnvelopeOrFail(t, conn)
	if response.GetRequestId() != 1 || response.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authentication response = %v, want player 7 for request 1", response)
	}
}

func writeClientEnvelope(t *testing.T, conn net.Conn, envelope *realtimepb.ClientEnvelope) {
	t.Helper()

	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal client envelope: %v", err)
	}
	if err := writeFrame(conn, payload); err != nil {
		t.Fatalf("write client envelope: %v", err)
	}
}

func readServerEnvelopeOrFail(t *testing.T, conn net.Conn) *realtimepb.ServerEnvelope {
	t.Helper()

	envelope, err := readServerEnvelope(conn)
	if err != nil {
		t.Fatalf("read server envelope: %v", err)
	}
	return envelope
}

func readServerEnvelopeWithDeadline(t *testing.T, conn net.Conn) *realtimepb.ServerEnvelope {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	defer func() {
		_ = conn.SetReadDeadline(time.Time{})
	}()
	return readServerEnvelopeOrFail(t, conn)
}

func samePlayerIDs(got, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

type fakeHandlerAuth struct {
	mu              sync.Mutex
	session         *auth.Session
	getSessionErr   error
	getSessionCalls int
}

func (f *fakeHandlerAuth) Register(context.Context, auth.RegisterInput) (*auth.AuthorizeResult, error) {
	return nil, errors.New("unexpected Register call")
}

func (f *fakeHandlerAuth) Login(context.Context, auth.LoginInput) (*auth.AuthorizeResult, error) {
	return nil, errors.New("unexpected Login call")
}

func (f *fakeHandlerAuth) Logout(context.Context, string) error {
	return errors.New("unexpected Logout call")
}

func (f *fakeHandlerAuth) GetSession(context.Context, string) (*auth.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getSessionCalls++
	return f.session, f.getSessionErr
}

func (f *fakeHandlerAuth) GetCurrentPlayer(context.Context, string) (*player.Player, error) {
	return nil, errors.New("unexpected GetCurrentPlayer call")
}

func (f *fakeHandlerAuth) getSessionCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getSessionCalls
}

type fakeHandlerPresence struct {
	mu               sync.Mutex
	getResult        *presence.Presence
	getErr           error
	getPlayerID      int64
	markOnlineErr    error
	markOnlineCalls  int
	markOfflineCalls int
	refreshCalls     int
}

type fakeHandlerMatch struct {
	startPlayerID  int64
	startWeapon    string
	startSolo      bool
	startResult    *rcenter.MatchResult
	startErr       error
	cancelPlayerID int64
	cancelErr      error
	resumePlayerID int64
	resumeResult   *rcenter.MatchResult
	resumeErr      error
	resumeFunc     func(context.Context, int64) (*rcenter.MatchResult, error)
}

type fakeHandlerFriend struct{ friendIDs []int64 }

func (f *fakeHandlerFriend) SendRequest(context.Context, int64, int64) error {
	return errors.New("unexpected SendRequest call")
}
func (f *fakeHandlerFriend) ListIncomingRequests(context.Context, int64) ([]*friend.Request, error) {
	return nil, errors.New("unexpected ListIncomingRequests call")
}
func (f *fakeHandlerFriend) ListOutgoingRequests(context.Context, int64) ([]*friend.Request, error) {
	return nil, errors.New("unexpected ListOutgoingRequests call")
}
func (f *fakeHandlerFriend) AcceptRequest(context.Context, int64, int64) error {
	return errors.New("unexpected AcceptRequest call")
}
func (f *fakeHandlerFriend) RejectRequest(context.Context, int64, int64) error {
	return errors.New("unexpected RejectRequest call")
}
func (f *fakeHandlerFriend) ListFriendIDs(context.Context, int64) ([]int64, error) {
	return f.friendIDs, nil
}
func (f *fakeHandlerFriend) DeleteFriend(context.Context, int64, int64) error {
	return errors.New("unexpected DeleteFriend call")
}

func (f *fakeHandlerMatch) Start(ctx context.Context, playerID int64, weapon string, solo bool) (*rcenter.MatchResult, error) {
	f.startPlayerID = playerID
	f.startWeapon = weapon
	f.startSolo = solo
	return f.startResult, f.startErr
}

func (f *fakeHandlerMatch) Cancel(_ context.Context, playerID int64) error {
	f.cancelPlayerID = playerID
	return f.cancelErr
}

func (f *fakeHandlerMatch) Resume(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	f.resumePlayerID = playerID
	if f.resumeFunc != nil {
		return f.resumeFunc(ctx, playerID)
	}
	if f.resumeResult == nil && f.resumeErr == nil {
		return nil, rcenter.ErrActiveMatchNotFound
	}
	return f.resumeResult, f.resumeErr
}

func (f *fakeHandlerPresence) MarkOnline(context.Context, int64, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markOnlineCalls++
	return f.markOnlineErr
}

func (f *fakeHandlerPresence) Get(_ context.Context, playerID int64) (*presence.Presence, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getPlayerID = playerID
	return f.getResult, f.getErr
}

type publishedHandlerRealtimeEvent struct {
	delivery statecontract.RealtimeDelivery
}

type fakeHandlerRealtimeClient struct {
	mu         sync.Mutex
	published  []publishedHandlerRealtimeEvent
	publishErr error
}

func (f *fakeHandlerRealtimeClient) PublishRealtime(_ context.Context, delivery *statecontract.RealtimeDelivery) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if delivery != nil {
		f.published = append(f.published, publishedHandlerRealtimeEvent{delivery: *delivery})
	}
	return f.publishErr
}

func (f *fakeHandlerRealtimeClient) SubscribeRealtime(context.Context, statecontract.RealtimeRoute) (<-chan *statecontract.RealtimeDelivery, error) {
	return nil, errors.New("unexpected SubscribeRealtime call")
}

func (f *fakeHandlerRealtimeClient) publishedEvent() (publishedHandlerRealtimeEvent, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.published) == 0 {
		return publishedHandlerRealtimeEvent{}, false
	}
	return f.published[0], true
}

type readDeadlineControlConn struct {
	net.Conn
	mu        sync.Mutex
	deadlines []time.Time
	setErr    error
	clearErr  error
}

func (c *readDeadlineControlConn) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.deadlines = append(c.deadlines, deadline)
	c.mu.Unlock()
	if deadline.IsZero() && c.clearErr != nil {
		return c.clearErr
	}
	if !deadline.IsZero() && c.setErr != nil {
		return c.setErr
	}
	return c.Conn.SetReadDeadline(deadline)
}

func (c *readDeadlineControlConn) readDeadlines() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Time(nil), c.deadlines...)
}

func waitForReadDeadlines(t *testing.T, conn *readDeadlineControlConn, count int) []time.Time {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		deadlines := conn.readDeadlines()
		if len(deadlines) >= count {
			return deadlines
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("SetReadDeadline calls = %d, want at least %d", len(conn.readDeadlines()), count)
	return nil
}

func (f *fakeHandlerPresence) MarkOffline(context.Context, int64, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markOfflineCalls++
	return nil
}

func (f *fakeHandlerPresence) Refresh(context.Context, int64, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshCalls++
	return nil
}

func (f *fakeHandlerPresence) onlineCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.markOnlineCalls
}

func (f *fakeHandlerPresence) offlineCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.markOfflineCalls
}

func (f *fakeHandlerPresence) refreshCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshCalls
}

var _ auth.Service = (*fakeHandlerAuth)(nil)
var _ friend.Service = (*fakeHandlerFriend)(nil)
var _ match.Service = (*fakeHandlerMatch)(nil)
var _ presence.Service = (*fakeHandlerPresence)(nil)
var _ statecontract.RealtimeClient = (*fakeHandlerRealtimeClient)(nil)
