package realtime

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"server/internal/contract/realtimepb"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/chat"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/leaderboard"
	"server/internal/logic/player"
	"server/internal/logic/presence"
)

func TestHandleSendWorldChatPublishesBroadcastDelivery(t *testing.T) {
	createdAt := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	chatService := &handlerMethodChatService{worldMessage: &chat.Message{
		MessageKey:       "message-1",
		ChannelType:      chat.ChannelWorld,
		ChannelKey:       statecontract.WorldChatChannelKey,
		SenderID:         7,
		Content:          "hello world",
		CreatedAt:        createdAt,
		ExpiresAt:        createdAt.Add(statecontract.WorldChatRetention),
		ClientMessageKey: "client-1",
	}}
	realtimeClient := &fakeHandlerRealtimeClient{publishErr: errors.New("publish unavailable")}
	handler := &Handler{chat: chatService, realtimeClient: realtimeClient}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleSendWorldChat(context.Background(), session, 7, &realtimepb.ClientEnvelope{
			RequestId: 12,
			Payload: &realtimepb.ClientEnvelope_ChatWorldSend{
				ChatWorldSend: &realtimepb.SendWorldChatRequest{Content: "hello world", ClientMessageKey: "client-1"},
			},
		})
	}, nil)

	if response.GetRequestId() != 12 || response.GetChatSent().GetMessage().GetMessageKey() != "message-1" || !keepConnection {
		t.Fatalf("world chat response = %v, keep = %v; want successful response", response, keepConnection)
	}
	if chatService.worldInput.SenderID != 7 || chatService.worldInput.Content != "hello world" || chatService.worldInput.ClientMessageKey != "client-1" {
		t.Fatalf("SendWorldMessage input = %+v, want sender 7 and complete request", chatService.worldInput)
	}
	published, ok := realtimeClient.publishedEvent()
	if !ok || published.delivery.Route.Type != statecontract.RealtimeRouteBroadcast || published.delivery.Route.ServerName != "" || published.delivery.Event == nil {
		t.Fatalf("published delivery = %+v, want broadcast delivery", published)
	}
	event := published.delivery.Event
	if event.Type != statecontract.RealtimeEventChatMessage || event.TargetPlayerID != 0 || event.ActorPlayerID != 7 || event.ChatMessage == nil || event.ChatMessage.MessageKey != "message-1" || event.ChatMessage.ChannelType != statecontract.ChatChannelWorld || event.ChatMessage.Content != "hello world" {
		t.Fatalf("published event = %+v, want complete world chat event", event)
	}
}

func TestHandleSendFriendRequestPushesLocalNotification(t *testing.T) {
	friendService := &handlerMethodFriendService{}
	handler := &Handler{friend: friendService, presence: &handlerMethodPresenceService{}, connManager: newConnectionManager()}
	targetServer, targetClient := net.Pipe()
	defer targetServer.Close()
	defer targetClient.Close()
	handler.connManager.Add(8, newSession(targetServer))

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleSendFriendRequest(context.Background(), session, 7, &realtimepb.ClientEnvelope{
			RequestId: 2,
			Payload: &realtimepb.ClientEnvelope_FriendRequestSend{
				FriendRequestSend: &realtimepb.SendFriendRequestRequest{ToPlayerId: 8},
			},
		})
	}, targetClient)

	if friendService.sendFrom != 7 || friendService.sendTo != 8 {
		t.Fatalf("SendRequest(from, to) = (%d, %d), want (7, 8)", friendService.sendFrom, friendService.sendTo)
	}
	if response.GetRequestId() != 2 || response.GetFriendRequestSent() == nil || !keepConnection {
		t.Fatalf("send response = %v, keep = %v; want success and an open connection", response, keepConnection)
	}
}

func TestHandleRejectFriendRequestUsesRejectPayload(t *testing.T) {
	friendService := &handlerMethodFriendService{}
	handler := &Handler{friend: friendService, presence: &handlerMethodPresenceService{}, connManager: newConnectionManager()}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleRejectFriendRequest(context.Background(), session, 7, &realtimepb.ClientEnvelope{
			RequestId: 3,
			Payload: &realtimepb.ClientEnvelope_FriendRequestReject{
				FriendRequestReject: &realtimepb.RejectFriendRequestRequest{FromPlayerId: 8},
			},
		})
	}, nil)

	if friendService.rejectFrom != 8 || friendService.rejectTo != 7 {
		t.Fatalf("RejectRequest(from, to) = (%d, %d), want (8, 7)", friendService.rejectFrom, friendService.rejectTo)
	}
	if response.GetFriendRequestHandledAck() == nil || !keepConnection {
		t.Fatalf("reject response = %v, keep = %v; want handled ack and an open connection", response, keepConnection)
	}
}

func TestHandleFriendRequestListsConvertDomainModels(t *testing.T) {
	createdAt := time.Date(2026, time.August, 11, 8, 0, 0, 0, time.UTC)
	friendService := &handlerMethodFriendService{
		incoming: []*friend.Request{{FromPlayerID: 8, ToPlayerID: 7, CreatedAt: createdAt}},
		outgoing: []*friend.Request{{FromPlayerID: 7, ToPlayerID: 9, CreatedAt: createdAt}},
	}
	handler := &Handler{friend: friendService, connManager: newConnectionManager()}

	incoming, keepIncoming := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleListIncomingFriendRequests(context.Background(), session, 7, &realtimepb.ClientEnvelope{RequestId: 4})
	}, nil)
	outgoing, keepOutgoing := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleListOutgoingFriendRequests(context.Background(), session, 7, &realtimepb.ClientEnvelope{RequestId: 5})
	}, nil)

	if len(incoming.GetFriendRequests().GetRequests()) != 1 || incoming.GetFriendRequests().GetRequests()[0].GetFromPlayerId() != 8 || !incoming.GetFriendRequests().GetRequests()[0].GetCreatedAt().AsTime().Equal(createdAt) || !keepIncoming {
		t.Fatalf("incoming response = %v, keep = %v; want request from player 8", incoming, keepIncoming)
	}
	if len(outgoing.GetFriendRequests().GetRequests()) != 1 || outgoing.GetFriendRequests().GetRequests()[0].GetToPlayerId() != 9 || !keepOutgoing {
		t.Fatalf("outgoing response = %v, keep = %v; want request to player 9", outgoing, keepOutgoing)
	}
}

func TestHandleAcceptAndDeleteFriendPassDomainArguments(t *testing.T) {
	friendService := &handlerMethodFriendService{}
	handler := &Handler{friend: friendService, presence: &handlerMethodPresenceService{}, connManager: newConnectionManager()}

	accepted, keepAccepted := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleAcceptFriendRequest(context.Background(), session, 7, &realtimepb.ClientEnvelope{
			RequestId: 6,
			Payload: &realtimepb.ClientEnvelope_FriendRequestAccept{
				FriendRequestAccept: &realtimepb.AcceptFriendRequestRequest{FromPlayerId: 8},
			},
		})
	}, nil)
	deleted, keepDeleted := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleDeleteFriend(context.Background(), session, 7, &realtimepb.ClientEnvelope{
			RequestId: 7,
			Payload: &realtimepb.ClientEnvelope_FriendDelete{
				FriendDelete: &realtimepb.DeleteFriendRequest{FriendPlayerId: 9},
			},
		})
	}, nil)

	if friendService.acceptFrom != 8 || friendService.acceptTo != 7 || accepted.GetFriendRequestHandledAck() == nil || !keepAccepted {
		t.Fatalf("accept args/response = (%d, %d, %v), want (8, 7, ack)", friendService.acceptFrom, friendService.acceptTo, accepted)
	}
	if friendService.deletePlayer != 7 || friendService.deleteFriend != 9 || deleted.GetFriendDeleted() == nil || !keepDeleted {
		t.Fatalf("delete args/response = (%d, %d, %v), want (7, 9, ack)", friendService.deletePlayer, friendService.deleteFriend, deleted)
	}
}

func TestHandleListFriendsReturnsInternalOnPresenceFailure(t *testing.T) {
	handler := &Handler{
		friend:      &handlerMethodFriendService{friendIDs: []int64{8}},
		player:      &handlerMethodPlayerService{result: &player.Player{ID: 8, Nickname: "Bob"}},
		presence:    &handlerMethodPresenceService{getErr: errors.New("state unavailable")},
		connManager: newConnectionManager(),
	}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleListFriends(context.Background(), session, 7, &realtimepb.ClientEnvelope{RequestId: 4})
	}, nil)

	if response.GetError().GetCode() != realtimepb.ErrorCode_INTERNAL || !keepConnection {
		t.Fatalf("friend list response = %v, keep = %v; want internal error and an open connection", response, keepConnection)
	}
}

func TestHandleListFriendsReturnsOfflineSummary(t *testing.T) {
	handler := &Handler{
		friend:      &handlerMethodFriendService{friendIDs: []int64{8}},
		player:      &handlerMethodPlayerService{result: &player.Player{ID: 8, Nickname: "Bob", Avatar: "bob.png"}},
		presence:    &handlerMethodPresenceService{getErr: presence.ErrNotFound},
		connManager: newConnectionManager(),
	}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleListFriends(context.Background(), session, 7, &realtimepb.ClientEnvelope{RequestId: 8})
	}, nil)

	friends := response.GetFriends().GetFriends()
	if len(friends) != 1 || friends[0].GetPlayerId() != 8 || friends[0].GetOnline() || friends[0].GetStatus() != presence.StatusOffline || !keepConnection {
		t.Fatalf("friend list response = %v, keep = %v; want offline player 8", response, keepConnection)
	}
}

func TestHandlePlayerAndGrowthResponses(t *testing.T) {
	growthValue := &growth.Growth{PlayerID: 7, AttackLevel: 2, AttackSpeedLevel: 3, HealthLevel: 4, MoveSpeedLevel: 5}
	handler := &Handler{
		player: &handlerMethodPlayerService{result: &player.Player{ID: 7, Nickname: "Alice", Coins: 900}},
		growth: &handlerMethodGrowthService{
			value:   growthValue,
			options: []growth.UpgradeOption{{Type: growth.UpgradeAttack, CurrentLevel: 2, NextCost: 100, MaxLevel: 10}},
			upgrade: &growth.UpgradeResult{Growth: growthValue, RemainingCoins: 800, Cost: 100},
		},
	}

	playerResponse, keepPlayer := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleGetPlayer(context.Background(), session, 7, &realtimepb.ClientEnvelope{RequestId: 9})
	}, nil)
	growthResponse, keepGrowth := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleGetGrowth(context.Background(), session, 7, &realtimepb.ClientEnvelope{RequestId: 10})
	}, nil)
	upgradeResponse, keepUpgrade := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleUpgradeGrowth(context.Background(), session, 7, &realtimepb.ClientEnvelope{
			RequestId: 11,
			Payload: &realtimepb.ClientEnvelope_GrowthUpgrade{
				GrowthUpgrade: &realtimepb.UpgradeGrowthRequest{Type: "attack"},
			},
		})
	}, nil)

	if playerResponse.GetPlayer().GetPlayer().GetCoins() != 900 || !keepPlayer {
		t.Fatalf("player response = %v, keep = %v; want 900 coins", playerResponse, keepPlayer)
	}
	if growthResponse.GetGrowth().GetGrowth().GetAttackLevel() != 2 || len(growthResponse.GetGrowth().GetGrowth().GetUpgradeOptions()) != 1 || !keepGrowth {
		t.Fatalf("growth response = %v, keep = %v; want converted growth", growthResponse, keepGrowth)
	}
	if upgradeResponse.GetGrowthUpgradeResult().GetRemainingCoins() != 800 || upgradeResponse.GetGrowthUpgradeResult().GetCost() != 100 || !keepUpgrade {
		t.Fatalf("upgrade response = %v, keep = %v; want balance 800 and cost 100", upgradeResponse, keepUpgrade)
	}
}

func TestHandleUpdatePlayerAvatar(t *testing.T) {
	playerService := &handlerMethodPlayerService{
		updateResult: &player.Player{
			ID:       7,
			Nickname: "Alice",
			Avatar:   string(player.AvatarMage),
			Email:    "alice@example.com",
			Phone:    "13800000000",
			Coins:    900,
		},
	}
	handler := &Handler{player: playerService}
	envelope := &realtimepb.ClientEnvelope{
		RequestId: 12,
		Payload: &realtimepb.ClientEnvelope_PlayerAvatarUpdate{
			PlayerAvatarUpdate: &realtimepb.PlayerAvatarUpdateRequest{Avatar: string(player.AvatarMage)},
		},
	}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleEnvelope(
			context.Background(),
			session,
			&auth.Session{PlayerID: 7},
			connectionID(0),
			envelope,
		)
	}, nil)

	got := response.GetPlayer().GetPlayer()
	if response.GetRequestId() != 12 || got.GetId() != 7 || got.GetNickname() != "Alice" ||
		got.GetAvatar() != string(player.AvatarMage) || got.GetEmail() != "alice@example.com" ||
		got.GetPhone() != "13800000000" || got.GetCoins() != 900 || !keepConnection {
		t.Fatalf("avatar update response = %v, keep = %v; want complete updated player", response, keepConnection)
	}
	if playerService.updatePlayerID != 7 || playerService.updateAvatar != string(player.AvatarMage) {
		t.Fatalf("UpdateAvatar input = (%d, %q), want (7, %q)", playerService.updatePlayerID, playerService.updateAvatar, player.AvatarMage)
	}
	if got := realtimeRequestType(envelope); got != "player" {
		t.Fatalf("realtimeRequestType() = %q, want player", got)
	}
}

func TestHandleUpdatePlayerAvatarMapsDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want realtimepb.ErrorCode
	}{
		{name: "invalid avatar", err: player.ErrInvalidAvatar, want: realtimepb.ErrorCode_INVALID_ARGUMENT},
		{name: "missing player", err: player.ErrNotFound, want: realtimepb.ErrorCode_NOT_FOUND},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{player: &handlerMethodPlayerService{updateErr: tt.err}}

			response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
				return handler.handleUpdatePlayerAvatar(context.Background(), session, 7, avatarUpdateEnvelope(13, string(player.AvatarMage)))
			}, nil)

			if response.GetRequestId() != 13 || response.GetError().GetCode() != tt.want || !keepConnection {
				t.Fatalf("avatar update response = %v, keep = %v; want error %v and open connection", response, keepConnection, tt.want)
			}
		})
	}
}

func TestHandleUpdatePlayerAvatarRejectsNilResult(t *testing.T) {
	handler := &Handler{player: &handlerMethodPlayerService{}}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleUpdatePlayerAvatar(context.Background(), session, 7, avatarUpdateEnvelope(14, string(player.AvatarMage)))
	}, nil)

	if response.GetRequestId() != 14 || response.GetError().GetCode() != realtimepb.ErrorCode_INTERNAL || !keepConnection {
		t.Fatalf("avatar update response = %v, keep = %v; want internal error and open connection", response, keepConnection)
	}
}

func TestHandleListLeaderboardDispatchesAndReturnsDuoEntries(t *testing.T) {
	leaderboardService := &handlerMethodLeaderboardService{result: &leaderboard.Result{
		Type:       leaderboard.TypeDuoClearTime,
		MapVersion: "wave-v1",
		Entries: []leaderboard.Entry{{
			Rank: 1,
			Players: []leaderboard.Player{
				{PlayerID: 7, Nickname: "Alice", Avatar: "alice.png"},
				{PlayerID: 8, Nickname: "Bob", Avatar: "bob.png"},
			},
			Score: 12_345,
		}},
	}}
	handler := &Handler{leaderboard: leaderboardService}
	envelope := &realtimepb.ClientEnvelope{
		RequestId: 15,
		Payload: &realtimepb.ClientEnvelope_LeaderboardList{
			LeaderboardList: &realtimepb.ListLeaderboardRequest{
				Type:       realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME,
				MapVersion: "wave-v1",
				Limit:      10,
			},
		},
	}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleEnvelope(context.Background(), session, &auth.Session{PlayerID: 7}, 0, envelope)
	}, nil)

	if leaderboardService.input.Type != leaderboard.TypeDuoClearTime || leaderboardService.input.MapVersion != "wave-v1" || leaderboardService.input.Limit != 10 {
		t.Fatalf("leaderboard input = %+v, want duo wave-v1 limit 10", leaderboardService.input)
	}
	entries := response.GetLeaderboard().GetEntries()
	if response.GetRequestId() != 15 || response.GetLeaderboard().GetType() != realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME ||
		response.GetLeaderboard().GetMapVersion() != "wave-v1" || len(entries) != 1 || len(entries[0].GetPlayers()) != 2 ||
		entries[0].GetPlayers()[0].GetPlayerId() != 7 || entries[0].GetPlayers()[1].GetPlayerId() != 8 || entries[0].GetScore() != 12_345 || !keepConnection {
		t.Fatalf("leaderboard response = %v, keep = %v; want converted duo entry and open connection", response, keepConnection)
	}
	if got := realtimeRequestType(envelope); got != "leaderboard" {
		t.Fatalf("realtimeRequestType() = %q, want leaderboard", got)
	}
}

func TestHandleListLeaderboardErrors(t *testing.T) {
	tests := []struct {
		name        string
		service     leaderboard.Service
		wantCode    realtimepb.ErrorCode
		wantMessage string
	}{
		{
			name:        "invalid query",
			service:     &handlerMethodLeaderboardService{err: leaderboard.ErrInvalidQuery},
			wantCode:    realtimepb.ErrorCode_INVALID_ARGUMENT,
			wantMessage: leaderboard.ErrInvalidQuery.Error(),
		},
		{
			name:        "internal service failure",
			service:     &handlerMethodLeaderboardService{err: errors.New("state unavailable")},
			wantCode:    realtimepb.ErrorCode_INTERNAL,
			wantMessage: "internal server error",
		},
		{
			name:        "nil result",
			service:     &handlerMethodLeaderboardService{},
			wantCode:    realtimepb.ErrorCode_INTERNAL,
			wantMessage: "unexpected leaderboard result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{leaderboard: tt.service}
			response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
				return handler.handleListLeaderboard(context.Background(), session, &realtimepb.ClientEnvelope{
					RequestId: 16,
					Payload: &realtimepb.ClientEnvelope_LeaderboardList{
						LeaderboardList: &realtimepb.ListLeaderboardRequest{},
					},
				})
			}, nil)

			if response.GetRequestId() != 16 || response.GetError().GetCode() != tt.wantCode ||
				response.GetError().GetMessage() != tt.wantMessage || !keepConnection {
				t.Fatalf("leaderboard response = %v, keep = %v; want %v %q and open connection", response, keepConnection, tt.wantCode, tt.wantMessage)
			}
		})
	}
}

func TestHandlerDomainErrorCodes(t *testing.T) {
	if got := growthErrorCode(growth.ErrInvalidUpgradeType); got != realtimepb.ErrorCode_INVALID_ARGUMENT {
		t.Fatalf("growthErrorCode(invalid type) = %v, want INVALID_ARGUMENT", got)
	}
	if got := growthErrorCode(growth.ErrInsufficientCoins); got != realtimepb.ErrorCode_CONFLICT {
		t.Fatalf("growthErrorCode(insufficient coins) = %v, want CONFLICT", got)
	}
	if got := growthErrorCode(growth.ErrGrowthNotFound); got != realtimepb.ErrorCode_NOT_FOUND {
		t.Fatalf("growthErrorCode(not found) = %v, want NOT_FOUND", got)
	}
	if got := playerErrorCode(player.ErrNotFound); got != realtimepb.ErrorCode_NOT_FOUND {
		t.Fatalf("playerErrorCode(not found) = %v, want NOT_FOUND", got)
	}
	if got := leaderboardErrorCode(leaderboard.ErrInvalidQuery); got != realtimepb.ErrorCode_INVALID_ARGUMENT {
		t.Fatalf("leaderboardErrorCode(invalid query) = %v, want INVALID_ARGUMENT", got)
	}
	if got := leaderboardErrorCode(errors.New("state unavailable")); got != realtimepb.ErrorCode_INTERNAL {
		t.Fatalf("leaderboardErrorCode(internal) = %v, want INTERNAL", got)
	}
}

func TestHandleLogoutDeletesSessionAndStopsConnection(t *testing.T) {
	authService := &handlerMethodAuthService{}
	handler := &Handler{auth: authService}

	response, keepConnection := invokeHandlerMethod(t, func(session *session) bool {
		return handler.handleLogout(context.Background(), session, &auth.Session{Token: "session-token", PlayerID: 7}, &realtimepb.ClientEnvelope{RequestId: 5})
	}, nil)

	if authService.logoutToken != "session-token" {
		t.Fatalf("Logout token = %q, want session-token", authService.logoutToken)
	}
	if response.GetLogoutAck() == nil || keepConnection {
		t.Fatalf("logout response = %v, keep = %v; want logout ack and a stopped connection", response, keepConnection)
	}
}

func invokeHandlerMethod(t *testing.T, invoke func(*session) bool, pushedConn net.Conn) (*realtimepb.ServerEnvelope, bool) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	result := make(chan bool, 1)
	go func() {
		result <- invoke(newSession(serverConn))
	}()

	if pushedConn != nil {
		pushed := readServerEnvelopeWithDeadline(t, pushedConn)
		if pushed.GetRequestId() != proactivePushRequestID || pushed.GetFriendRequestReceived().GetPlayerId() != 7 {
			t.Fatalf("local push = %v, want friend request notification from player 7", pushed)
		}
	}
	response := readServerEnvelopeWithDeadline(t, clientConn)
	return response, <-result
}

type handlerMethodFriendService struct {
	friendIDs    []int64
	incoming     []*friend.Request
	outgoing     []*friend.Request
	sendFrom     int64
	sendTo       int64
	acceptFrom   int64
	acceptTo     int64
	rejectFrom   int64
	rejectTo     int64
	deletePlayer int64
	deleteFriend int64
}

func (f *handlerMethodFriendService) SendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.sendFrom, f.sendTo = fromPlayerID, toPlayerID
	return nil
}
func (f *handlerMethodFriendService) ListIncomingRequests(context.Context, int64) ([]*friend.Request, error) {
	return f.incoming, nil
}
func (f *handlerMethodFriendService) ListOutgoingRequests(context.Context, int64) ([]*friend.Request, error) {
	return f.outgoing, nil
}
func (f *handlerMethodFriendService) AcceptRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.acceptFrom, f.acceptTo = fromPlayerID, toPlayerID
	return nil
}
func (f *handlerMethodFriendService) RejectRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.rejectFrom, f.rejectTo = fromPlayerID, toPlayerID
	return nil
}
func (f *handlerMethodFriendService) ListFriendIDs(context.Context, int64) ([]int64, error) {
	return f.friendIDs, nil
}
func (f *handlerMethodFriendService) DeleteFriend(_ context.Context, playerID, friendID int64) error {
	f.deletePlayer, f.deleteFriend = playerID, friendID
	return nil
}

type handlerMethodPlayerService struct {
	result         *player.Player
	err            error
	updatePlayerID int64
	updateAvatar   string
	updateResult   *player.Player
	updateErr      error
}

func (f *handlerMethodPlayerService) Create(context.Context, player.CreateInput) (*player.Player, error) {
	return nil, errors.New("unexpected Create call")
}
func (f *handlerMethodPlayerService) Get(context.Context, int64) (*player.Player, error) {
	return f.result, f.err
}
func (f *handlerMethodPlayerService) UpdateAvatar(_ context.Context, playerID int64, avatar string) (*player.Player, error) {
	f.updatePlayerID = playerID
	f.updateAvatar = avatar
	return f.updateResult, f.updateErr
}

type handlerMethodPresenceService struct {
	result *presence.Presence
	getErr error
}

type handlerMethodGrowthService struct {
	value   *growth.Growth
	options []growth.UpgradeOption
	upgrade *growth.UpgradeResult
}

type handlerMethodLeaderboardService struct {
	input  leaderboard.ListInput
	result *leaderboard.Result
	err    error
}

func (f *handlerMethodLeaderboardService) List(_ context.Context, input leaderboard.ListInput) (*leaderboard.Result, error) {
	f.input = input
	return f.result, f.err
}

func (f *handlerMethodGrowthService) Upgrade(context.Context, int64, growth.UpgradeType) (*growth.UpgradeResult, error) {
	return f.upgrade, nil
}
func (f *handlerMethodGrowthService) Get(context.Context, int64) (*growth.Growth, error) {
	return f.value, nil
}
func (f *handlerMethodGrowthService) UpgradeOptions(*growth.Growth) ([]growth.UpgradeOption, error) {
	return f.options, nil
}

func (f *handlerMethodPresenceService) MarkOnline(context.Context, int64, string) error  { return nil }
func (f *handlerMethodPresenceService) MarkOffline(context.Context, int64, string) error { return nil }
func (f *handlerMethodPresenceService) Refresh(context.Context, int64, string) error     { return nil }
func (f *handlerMethodPresenceService) Get(context.Context, int64) (*presence.Presence, error) {
	return f.result, f.getErr
}

type handlerMethodAuthService struct {
	logoutToken string
}

type handlerMethodChatService struct {
	worldInput   chat.SendWorldMessageInput
	worldMessage *chat.Message
}

func (f *handlerMethodChatService) SendWorldMessage(_ context.Context, input chat.SendWorldMessageInput) (*chat.Message, error) {
	f.worldInput = input
	return f.worldMessage, nil
}
func (f *handlerMethodChatService) SendDirectMessage(context.Context, chat.SendDirectMessageInput) (*chat.Message, error) {
	return nil, errors.New("unexpected SendDirectMessage call")
}
func (f *handlerMethodChatService) ListWorldMessages(context.Context, chat.ListWorldMessagesInput) ([]*chat.Message, error) {
	return nil, errors.New("unexpected ListWorldMessages call")
}
func (f *handlerMethodChatService) ListDirectMessages(context.Context, chat.ListDirectMessagesInput) ([]*chat.Message, error) {
	return nil, errors.New("unexpected ListDirectMessages call")
}

func (f *handlerMethodAuthService) Register(context.Context, auth.RegisterInput) (*auth.AuthorizeResult, error) {
	return nil, errors.New("unexpected Register call")
}
func (f *handlerMethodAuthService) Login(context.Context, auth.LoginInput) (*auth.AuthorizeResult, error) {
	return nil, errors.New("unexpected Login call")
}
func (f *handlerMethodAuthService) Logout(_ context.Context, token string) error {
	f.logoutToken = token
	return nil
}
func (f *handlerMethodAuthService) GetSession(context.Context, string) (*auth.Session, error) {
	return nil, errors.New("unexpected GetSession call")
}
func (f *handlerMethodAuthService) GetCurrentPlayer(context.Context, string) (*player.Player, error) {
	return nil, errors.New("unexpected GetCurrentPlayer call")
}

var _ friend.Service = (*handlerMethodFriendService)(nil)
var _ player.Service = (*handlerMethodPlayerService)(nil)
var _ presence.Service = (*handlerMethodPresenceService)(nil)
var _ growth.Service = (*handlerMethodGrowthService)(nil)
var _ leaderboard.Service = (*handlerMethodLeaderboardService)(nil)
var _ auth.Service = (*handlerMethodAuthService)(nil)
var _ chat.Service = (*handlerMethodChatService)(nil)

func avatarUpdateEnvelope(requestID uint64, avatar string) *realtimepb.ClientEnvelope {
	return &realtimepb.ClientEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ClientEnvelope_PlayerAvatarUpdate{
			PlayerAvatarUpdate: &realtimepb.PlayerAvatarUpdateRequest{Avatar: avatar},
		},
	}
}
