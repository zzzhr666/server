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
	if published.serverName != "logic-old" || published.event.Type != statecontract.RealtimeEventConnectionReplaced || published.event.TargetPlayerID != 7 || published.event.ActorPlayerID != 7 {
		t.Fatalf("published event = %+v, want replacement for player 7 on logic-old", published)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
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
	if !ok || presenceService.getPlayerID != 8 || published.serverName != "logic-friend" || published.event.TargetPlayerID != 8 || published.event.ActorPlayerID != 7 || !published.event.Online {
		t.Fatalf("published friend presence = %+v, want online event for friend 8 on logic-friend", published)
	}
}

func TestHandlerServeSessionStartsMatch(t *testing.T) {
	matchService := &fakeHandlerMatch{startResult: &rcenter.MatchResult{Status: rcenter.MatchStatusWaiting}}
	handler := newHandlerWithMatch(matchService)
	_, clientConn, done := startHandlerSession(t, handler)
	defer clientConn.Close()

	authenticateSession(t, clientConn)
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "axe"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetRequestId() != 2 || response.GetMatchResult().GetStatus() != string(rcenter.MatchStatusWaiting) {
		t.Fatalf("match response = %v, want waiting result for request 2", response)
	}
	if matchService.startPlayerID != 7 || matchService.startWeapon != "axe" {
		t.Fatalf("Start(playerID, weapon) = (%d, %q), want (7, %q)", matchService.startPlayerID, matchService.startWeapon, "axe")
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
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "axe"))
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
	if presenceService.getPlayerID != 8 || published.serverName != "logic-remote" || published.event.Type != statecontract.RealtimeEventMatchResult || published.event.TargetPlayerID != 8 || published.event.ActorPlayerID != 7 || published.event.MatchStatus != string(rcenter.MatchStatusMatched) || published.event.RoomName != "room-7-8" || published.event.MatchToken != "match-token" || published.event.BattleNodeName != "battle-1" || published.event.BattleUDPAddr != "127.0.0.1:9001" || !samePlayerIDs(published.event.MatchPlayerIDs, []int64{7, 8}) {
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
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "axe"))
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
	writeClientEnvelope(t, clientConn, matchStartEnvelope(2, "wand"))
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

func matchStartEnvelope(requestID uint64, weapon string) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_MatchStart{
			MatchStart: &realtimepb.MatchStartRequest{Weapon: weapon},
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

func (f *fakeHandlerMatch) Start(_ context.Context, playerID int64, weapon string) (*rcenter.MatchResult, error) {
	f.startPlayerID = playerID
	f.startWeapon = weapon
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
	serverName string
	event      statecontract.RealtimeEvent
}

type fakeHandlerRealtimeClient struct {
	mu         sync.Mutex
	published  []publishedHandlerRealtimeEvent
	publishErr error
}

func (f *fakeHandlerRealtimeClient) PublishRealtimeToServer(_ context.Context, serverName string, event *statecontract.RealtimeEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if event != nil {
		f.published = append(f.published, publishedHandlerRealtimeEvent{serverName: serverName, event: *event})
	}
	return f.publishErr
}

func (f *fakeHandlerRealtimeClient) SubscribeRealtime(context.Context, string) (<-chan *statecontract.RealtimeEvent, error) {
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
