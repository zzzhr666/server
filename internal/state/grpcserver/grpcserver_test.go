package grpcserver

import (
	"context"
	"testing"
	"time"

	statecontract "server/internal/contract/state"
	"server/internal/contract/statepb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetAccount(t *testing.T) {
	state := &fakeStateClient{
		account: &statecontract.Account{
			Username:     "alice",
			PasswordHash: "hash",
			PlayerID:     7,
		},
	}
	server := newTestServer(state)

	res, err := server.GetAccount(context.Background(), &statepb.GetAccountRequest{Username: "alice"})
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if state.gotUsername != "alice" {
		t.Fatalf("state got username = %q, want alice", state.gotUsername)
	}
	if res.GetAccount().GetPlayerId() != 7 {
		t.Fatalf("player id = %d, want 7", res.GetAccount().GetPlayerId())
	}
}

func TestGetAccountNotFound(t *testing.T) {
	server := newTestServer(&fakeStateClient{err: statecontract.ErrAccountNotFound})

	_, err := server.GetAccount(context.Background(), &statepb.GetAccountRequest{Username: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetAccount code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestCreateAccountAlreadyExists(t *testing.T) {
	server := newTestServer(&fakeStateClient{err: statecontract.ErrAccountExists})

	_, err := server.CreateAccount(context.Background(), &statepb.CreateAccountRequest{
		Account: &statepb.Account{Username: "alice"},
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("CreateAccount code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
}

func TestRegisterAccount(t *testing.T) {
	expiresAt := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	state := &fakeStateClient{
		registerResult: &statecontract.RegisterAccountResult{
			Account: &statecontract.Account{Username: "alice", PasswordHash: "hash", PlayerID: 7},
			Player:  &statecontract.Player{ID: 7, Nickname: "Alice", Avatar: "avatar", Email: "a@example.com", Phone: "123"},
			Session: &statecontract.Session{Token: "token-1", PlayerID: 7, ExpiresAt: expiresAt},
		},
	}
	server := newTestServer(state)

	res, err := server.RegisterAccount(context.Background(), &statepb.RegisterAccountRequest{
		Username:         "alice",
		PasswordHash:     "hash",
		Nickname:         "Alice",
		Avatar:           "avatar",
		Email:            "a@example.com",
		Phone:            "123",
		SessionToken:     "token-1",
		SessionExpiresAt: timestamppb.New(expiresAt),
	})
	if err != nil {
		t.Fatalf("RegisterAccount returned error: %v", err)
	}
	if state.registerInput.Username != "alice" {
		t.Fatalf("register username = %q, want alice", state.registerInput.Username)
	}
	if !state.registerInput.SessionExpiresAt.Equal(expiresAt) {
		t.Fatalf("session expires at = %v, want %v", state.registerInput.SessionExpiresAt, expiresAt)
	}
	if res.GetPlayer().GetNickname() != "Alice" {
		t.Fatalf("player nickname = %q, want Alice", res.GetPlayer().GetNickname())
	}
	if !res.GetSession().GetExpiresAt().AsTime().Equal(expiresAt) {
		t.Fatalf("response session expires at = %v, want %v", res.GetSession().GetExpiresAt().AsTime(), expiresAt)
	}
}

func TestSessionMethods(t *testing.T) {
	expiresAt := time.Date(2026, 7, 19, 13, 0, 0, 0, time.UTC)
	state := &fakeStateClient{
		session: &statecontract.Session{Token: "token-1", PlayerID: 7, ExpiresAt: expiresAt},
	}
	server := newTestServer(state)

	_, err := server.CreateSession(context.Background(), &statepb.CreateSessionRequest{
		Session: &statepb.Session{Token: "token-1", PlayerId: 7, ExpiresAt: timestamppb.New(expiresAt)},
	})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if state.createdSession.Token != "token-1" {
		t.Fatalf("created session token = %q, want token-1", state.createdSession.Token)
	}

	getRes, err := server.GetSession(context.Background(), &statepb.GetSessionRequest{Token: "token-1"})
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if getRes.GetSession().GetPlayerId() != 7 {
		t.Fatalf("session player id = %d, want 7", getRes.GetSession().GetPlayerId())
	}

	_, err = server.DeleteSession(context.Background(), &statepb.DeleteSessionRequest{Token: "token-1"})
	if err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
	if state.deletedToken != "token-1" {
		t.Fatalf("deleted token = %q, want token-1", state.deletedToken)
	}
}

func TestPlayerMethods(t *testing.T) {
	state := &fakeStateClient{
		player:       &statecontract.Player{ID: 7, Nickname: "Alice"},
		nextPlayerID: 8,
	}
	server := newTestServer(state)

	_, err := server.CreatePlayer(context.Background(), &statepb.CreatePlayerRequest{
		Player: &statepb.Player{Id: 7, Nickname: "Alice"},
	})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	if state.createdPlayer.ID != 7 {
		t.Fatalf("created player id = %d, want 7", state.createdPlayer.ID)
	}

	getRes, err := server.GetPlayer(context.Background(), &statepb.GetPlayerRequest{Id: 7})
	if err != nil {
		t.Fatalf("GetPlayer returned error: %v", err)
	}
	if getRes.GetPlayer().GetNickname() != "Alice" {
		t.Fatalf("player nickname = %q, want Alice", getRes.GetPlayer().GetNickname())
	}

	updateRes, err := server.UpdatePlayerAvatar(context.Background(), &statepb.UpdatePlayerAvatarRequest{
		PlayerId: 7,
		Avatar:   "mage",
	})
	if err != nil {
		t.Fatalf("UpdatePlayerAvatar returned error: %v", err)
	}
	if state.updatedPlayerAvatarID != 7 || state.updatedAvatar != "mage" {
		t.Fatalf("update avatar input = (%d, %q), want (7, mage)", state.updatedPlayerAvatarID, state.updatedAvatar)
	}
	if updateRes.GetPlayer().GetId() != 7 || updateRes.GetPlayer().GetAvatar() != "mage" {
		t.Fatalf("updated player = %+v, want id 7 avatar mage", updateRes.GetPlayer())
	}

	nextRes, err := server.NextPlayerID(context.Background(), &statepb.NextPlayerIDRequest{})
	if err != nil {
		t.Fatalf("NextPlayerID returned error: %v", err)
	}
	if nextRes.GetId() != 8 {
		t.Fatalf("next player id = %d, want 8", nextRes.GetId())
	}
}

func TestUpdatePlayerAvatarMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "invalid input", err: statecontract.ErrInvalidPlayer, want: codes.InvalidArgument},
		{name: "missing player", err: statecontract.ErrPlayerNotFound, want: codes.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(&fakeStateClient{err: tt.err})
			_, err := server.UpdatePlayerAvatar(context.Background(), &statepb.UpdatePlayerAvatarRequest{})
			if status.Code(err) != tt.want {
				t.Fatalf("UpdatePlayerAvatar code = %v, want %v", status.Code(err), tt.want)
			}
		})
	}
}

func TestSettleMatchRewards(t *testing.T) {
	state := &fakeStateClient{
		settleMatchRewardsResult: &statecontract.SettleMatchRewardsResult{Applied: true},
	}
	server := newTestServer(state)

	res, err := server.SettleMatchRewards(context.Background(), &statepb.SettleMatchRewardRequest{
		SettlementId: "room-1",
		Rewards: []*statepb.PlayerCoinReward{
			{PlayerId: 7, Amount: 50},
			{PlayerId: 8, Amount: 30},
		},
		Leaderboard: &statepb.MatchLeaderboardRecord{
			Mode:             "duo",
			MapVersion:       "wave-v1",
			Cleared:          true,
			CombatDurationMs: 83_250,
			Players: []*statepb.PlayerLeaderboardRecord{
				{PlayerId: 7, TotalKills: 13},
				{PlayerId: 8, TotalKills: 17},
			},
		},
	})
	if err != nil {
		t.Fatalf("SettleMatchRewards returned error: %v", err)
	}
	if state.settleMatchRewardsInput.SettlementID != "room-1" {
		t.Fatalf("settlement id = %q, want room-1", state.settleMatchRewardsInput.SettlementID)
	}
	if len(state.settleMatchRewardsInput.Rewards) != 2 || state.settleMatchRewardsInput.Rewards[0].PlayerID != 7 || state.settleMatchRewardsInput.Rewards[1].Amount != 30 {
		t.Fatalf("settlement rewards = %+v, want converted request rewards", state.settleMatchRewardsInput.Rewards)
	}
	leaderboard := state.settleMatchRewardsInput.Leaderboard
	if leaderboard == nil || leaderboard.Mode != "duo" || leaderboard.MapVersion != "wave-v1" || !leaderboard.Cleared || leaderboard.CombatDurationMS != 83_250 {
		t.Fatalf("settlement leaderboard = %+v, want converted metadata", leaderboard)
	}
	if len(leaderboard.Players) != 2 || leaderboard.Players[0].PlayerID != 7 || leaderboard.Players[0].TotalKills != 13 || leaderboard.Players[1].PlayerID != 8 || leaderboard.Players[1].TotalKills != 17 {
		t.Fatalf("settlement leaderboard players = %+v, want converted players", leaderboard.Players)
	}
	if !res.GetApplied() {
		t.Fatal("response applied = false, want true")
	}
}

func TestSettleMatchRewardsInvalidSettlement(t *testing.T) {
	server := newTestServer(&fakeStateClient{err: statecontract.ErrInvalidSettlement})

	_, err := server.SettleMatchRewards(context.Background(), &statepb.SettleMatchRewardRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("SettleMatchRewards code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestChatMessages(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	expiresAt := createdAt.Add(statecontract.DirectChatRetention)
	state := &fakeStateClient{
		savedChatMessage: &statecontract.ChatMessage{
			MessageKey:       "msg-1",
			ChannelType:      statecontract.ChatChannelDirect,
			ChannelKey:       "direct:7:8",
			SenderID:         7,
			ReceiverID:       8,
			Content:          "hello",
			CreatedAt:        createdAt,
			ExpiresAt:        expiresAt,
			ClientMessageKey: "client-msg-1",
		},
		chatMessages: []*statecontract.ChatMessage{
			{
				MessageKey:       "msg-1",
				ChannelType:      statecontract.ChatChannelDirect,
				ChannelKey:       "direct:7:8",
				SenderID:         7,
				ReceiverID:       8,
				Content:          "hello",
				CreatedAt:        createdAt,
				ExpiresAt:        expiresAt,
				ClientMessageKey: "client-msg-1",
			},
		},
	}
	server := newTestServer(state)

	saveRes, err := server.SaveChatMessage(context.Background(), &statepb.SaveChatMessageRequest{
		ChannelType:      string(statecontract.ChatChannelDirect),
		ChannelKey:       "direct:7:8",
		SenderId:         7,
		ReceiverId:       8,
		Content:          "hello",
		CreatedAt:        timestamppb.New(createdAt),
		ExpiresAt:        timestamppb.New(expiresAt),
		MaxMessages:      statecontract.DirectChatMaxMessages,
		ClientMessageKey: "client-msg-1",
	})
	if err != nil {
		t.Fatalf("SaveChatMessage returned error: %v", err)
	}
	if state.saveChatMessageInput.ChannelKey != "direct:7:8" {
		t.Fatalf("save chat channel key = %q, want direct:7:8", state.saveChatMessageInput.ChannelKey)
	}
	if state.saveChatMessageInput.MaxMessages != statecontract.DirectChatMaxMessages {
		t.Fatalf("save chat max messages = %d, want %d", state.saveChatMessageInput.MaxMessages, statecontract.DirectChatMaxMessages)
	}
	if saveRes.GetMessage().GetMessageKey() != "msg-1" {
		t.Fatalf("saved message key = %q, want msg-1", saveRes.GetMessage().GetMessageKey())
	}

	listRes, err := server.ListChatMessages(context.Background(), &statepb.ListChatMessagesRequest{
		ChannelType:      string(statecontract.ChatChannelDirect),
		ChannelKey:       "direct:7:8",
		Limit:            20,
		BeforeMessageKey: "msg-0",
	})
	if err != nil {
		t.Fatalf("ListChatMessages returned error: %v", err)
	}
	if state.listChatMessagesInput.BeforeMessageKey != "msg-0" {
		t.Fatalf("list chat before message key = %q, want msg-0", state.listChatMessagesInput.BeforeMessageKey)
	}
	if len(listRes.GetMessages()) != 1 || listRes.GetMessages()[0].GetMessageKey() != "msg-1" {
		t.Fatalf("chat messages = %+v, want one saved message", listRes.GetMessages())
	}
}

func TestListLeaderboard(t *testing.T) {
	state := &fakeStateClient{
		listLeaderboardResult: &statecontract.ListLeaderboardResult{
			Type:       statecontract.LeaderboardTypeDuoClearTime,
			MapVersion: "wave-v1",
			Entries: []statecontract.LeaderboardEntry{
				{
					Rank:  1,
					Score: 82_000,
					Players: []statecontract.LeaderboardPlayer{
						{PlayerID: 7, Nickname: "Alice", Avatar: "mage"},
						{PlayerID: 8, Nickname: "Bob", Avatar: "knight"},
					},
				},
			},
		},
	}
	server := newTestServer(state)

	response, err := server.ListLeaderboard(context.Background(), &statepb.ListLeaderboardRequest{
		Type:       statepb.LeaderboardType_DUO_CLEAR_TIME,
		MapVersion: "wave-v1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("ListLeaderboard returned error: %v", err)
	}
	if state.listLeaderboardInput.Type != statecontract.LeaderboardTypeDuoClearTime || state.listLeaderboardInput.MapVersion != "wave-v1" || state.listLeaderboardInput.Limit != 10 {
		t.Fatalf("ListLeaderboard input = %+v, want duo wave-v1 limit 10", state.listLeaderboardInput)
	}
	if response.GetType() != statepb.LeaderboardType_DUO_CLEAR_TIME || response.GetMapVersion() != "wave-v1" || len(response.GetEntries()) != 1 {
		t.Fatalf("ListLeaderboard response = %+v, want converted result", response)
	}
	players := response.GetEntries()[0].GetPlayers()
	if len(players) != 2 || players[0].GetNickname() != "Alice" || players[0].GetAvatar() != "mage" || players[1].GetPlayerId() != 8 {
		t.Fatalf("ListLeaderboard players = %+v, want converted duo", players)
	}
}

func TestListLeaderboardInvalidQuery(t *testing.T) {
	server := newTestServer(&fakeStateClient{err: statecontract.ErrInvalidLeaderboardQuery})

	_, err := server.ListLeaderboard(context.Background(), &statepb.ListLeaderboardRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListLeaderboard code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestPresenceMethods(t *testing.T) {
	updatedAt := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	state := &fakeStateClient{
		presence: &statecontract.Presence{
			PlayerID:   7,
			ServerName: "logic-1",
			Status:     "online",
			UpdatedAt:  updatedAt,
		},
	}
	server := newTestServer(state)

	_, err := server.SetPresence(context.Background(), &statepb.SetPresenceRequest{
		Presence: &statepb.Presence{
			PlayerId:   7,
			ServerName: "logic-1",
			Status:     "online",
			UpdatedAt:  timestamppb.New(updatedAt),
		},
	})
	if err != nil {
		t.Fatalf("SetPresence returned error: %v", err)
	}
	if state.setPresence.PlayerID != 7 {
		t.Fatalf("set presence player id = %d, want 7", state.setPresence.PlayerID)
	}
	if state.setPresence.Status != "online" {
		t.Fatalf("set presence status = %q, want online", state.setPresence.Status)
	}

	getRes, err := server.GetPresence(context.Background(), &statepb.GetPresenceRequest{PlayerId: 7})
	if err != nil {
		t.Fatalf("GetPresence returned error: %v", err)
	}
	if getRes.GetPresence().GetServerName() != "logic-1" {
		t.Fatalf("presence server name = %q, want logic-1", getRes.GetPresence().GetServerName())
	}

	_, err = server.ClearPresence(context.Background(), &statepb.ClearPresenceRequest{PlayerId: 7, ServerName: "logic-1"})
	if err != nil {
		t.Fatalf("ClearPresence returned error: %v", err)
	}
	if state.clearedPlayerID != 7 {
		t.Fatalf("cleared player id = %d, want 7", state.clearedPlayerID)
	}
	if state.clearedServerName != "logic-1" {
		t.Fatalf("cleared server name = %q, want logic-1", state.clearedServerName)
	}

	refreshedAt := updatedAt.Add(time.Minute)
	_, err = server.RefreshPresence(context.Background(), &statepb.RefreshPresenceRequest{
		PlayerId:   7,
		ServerName: "logic-1",
		UpdatedAt:  timestamppb.New(refreshedAt),
		Ttl:        durationpb.New(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("RefreshPresence returned error: %v", err)
	}
	if state.refreshedPlayerID != 7 {
		t.Fatalf("refreshed player id = %d, want 7", state.refreshedPlayerID)
	}
	if state.refreshedServerName != "logic-1" {
		t.Fatalf("refreshed server name = %q, want logic-1", state.refreshedServerName)
	}
	if !state.refreshedAt.Equal(refreshedAt) {
		t.Fatalf("refreshed at = %v, want %v", state.refreshedAt, refreshedAt)
	}
	if state.refreshedTTL != 2*time.Minute {
		t.Fatalf("refreshed ttl = %v, want %v", state.refreshedTTL, 2*time.Minute)
	}
}

func TestGetPresenceNotFound(t *testing.T) {
	server := newTestServer(&fakeStateClient{err: statecontract.ErrPresenceNotFound})

	_, err := server.GetPresence(context.Background(), &statepb.GetPresenceRequest{PlayerId: 7})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetPresence code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestSetPresenceInvalidArgument(t *testing.T) {
	server := newTestServer(&fakeStateClient{err: statecontract.ErrInvalidPresence})

	_, err := server.SetPresence(context.Background(), &statepb.SetPresenceRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("SetPresence code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestFriendMethods(t *testing.T) {
	createdAt := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	state := &fakeStateClient{
		incomingFriendRequests: []*statecontract.FriendRequest{
			{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: createdAt},
		},
		outgoingFriendRequests: []*statecontract.FriendRequest{
			{FromPlayerID: 8, ToPlayerID: 9, CreatedAt: createdAt.Add(time.Minute)},
		},
		friendIDs: []int64{10, 11},
	}
	server := newTestServer(state)

	_, err := server.SendFriendRequest(context.Background(), &statepb.SendFriendRequestRequest{
		FromPlayerId: 7,
		ToPlayerId:   8,
	})
	if err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}
	if state.sentFriendRequestFrom != 7 {
		t.Fatalf("sent friend request from = %d, want 7", state.sentFriendRequestFrom)
	}
	if state.sentFriendRequestTo != 8 {
		t.Fatalf("sent friend request to = %d, want 8", state.sentFriendRequestTo)
	}

	incomingRes, err := server.ListIncomingRequest(context.Background(), &statepb.ListFriendRequestRequest{PlayerId: 8})
	if err != nil {
		t.Fatalf("ListIncomingRequest returned error: %v", err)
	}
	if state.listIncomingFriendRequestsPlayerID != 8 {
		t.Fatalf("incoming player id = %d, want 8", state.listIncomingFriendRequestsPlayerID)
	}
	if len(incomingRes.GetRequests()) != 1 {
		t.Fatalf("incoming requests = %d, want 1", len(incomingRes.GetRequests()))
	}
	if incomingRes.GetRequests()[0].GetFromPlayer() != 7 {
		t.Fatalf("incoming from player = %d, want 7", incomingRes.GetRequests()[0].GetFromPlayer())
	}
	if !incomingRes.GetRequests()[0].GetCreatedAt().AsTime().Equal(createdAt) {
		t.Fatalf("incoming created at = %v, want %v", incomingRes.GetRequests()[0].GetCreatedAt().AsTime(), createdAt)
	}

	outgoingRes, err := server.ListOutgoingRequest(context.Background(), &statepb.ListFriendRequestRequest{PlayerId: 8})
	if err != nil {
		t.Fatalf("ListOutgoingRequest returned error: %v", err)
	}
	if state.listOutgoingFriendRequestsPlayerID != 8 {
		t.Fatalf("outgoing player id = %d, want 8", state.listOutgoingFriendRequestsPlayerID)
	}
	if len(outgoingRes.GetRequests()) != 1 {
		t.Fatalf("outgoing requests = %d, want 1", len(outgoingRes.GetRequests()))
	}
	if outgoingRes.GetRequests()[0].GetToPlayer() != 9 {
		t.Fatalf("outgoing to player = %d, want 9", outgoingRes.GetRequests()[0].GetToPlayer())
	}

	_, err = server.AcceptFriendRequest(context.Background(), &statepb.HandleFriendRequestRequest{
		FromPlayerId: 7,
		ToPlayerId:   8,
	})
	if err != nil {
		t.Fatalf("AcceptFriendRequest returned error: %v", err)
	}
	if state.acceptedFriendRequestFrom != 7 {
		t.Fatalf("accepted from player = %d, want 7", state.acceptedFriendRequestFrom)
	}
	if state.acceptedFriendRequestTo != 8 {
		t.Fatalf("accepted to player = %d, want 8", state.acceptedFriendRequestTo)
	}

	_, err = server.RejectFriendRequest(context.Background(), &statepb.HandleFriendRequestRequest{
		FromPlayerId: 9,
		ToPlayerId:   8,
	})
	if err != nil {
		t.Fatalf("RejectFriendRequest returned error: %v", err)
	}
	if state.rejectedFriendRequestFrom != 9 {
		t.Fatalf("rejected from player = %d, want 9", state.rejectedFriendRequestFrom)
	}
	if state.rejectedFriendRequestTo != 8 {
		t.Fatalf("rejected to player = %d, want 8", state.rejectedFriendRequestTo)
	}

	friendIDsRes, err := server.ListFriendIDs(context.Background(), &statepb.ListFriendIDsRequest{PlayerId: 7})
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	if state.listFriendIDsPlayerID != 7 {
		t.Fatalf("list friend ids player id = %d, want 7", state.listFriendIDsPlayerID)
	}
	if len(friendIDsRes.GetFriendPlayerIds()) != 2 || friendIDsRes.GetFriendPlayerIds()[0] != 10 || friendIDsRes.GetFriendPlayerIds()[1] != 11 {
		t.Fatalf("friend ids = %v, want [10 11]", friendIDsRes.GetFriendPlayerIds())
	}

	_, err = server.DeleteFriend(context.Background(), &statepb.DeleteFriendRequest{
		PlayerId:       7,
		FriendPlayerId: 10,
	})
	if err != nil {
		t.Fatalf("DeleteFriend returned error: %v", err)
	}
	if state.deletedFriendPlayerID != 7 {
		t.Fatalf("delete friend player id = %d, want 7", state.deletedFriendPlayerID)
	}
	if state.deletedFriendFriendPlayerID != 10 {
		t.Fatalf("delete friend friend player id = %d, want 10", state.deletedFriendFriendPlayerID)
	}
}

func TestFriendErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{
			name: "friend request not found",
			err:  statecontract.ErrFriendRequestNotFound,
			want: codes.NotFound,
		},
		{
			name: "friend not found",
			err:  statecontract.ErrFriendNotFound,
			want: codes.NotFound,
		},
		{
			name: "friend request exists",
			err:  statecontract.ErrFriendRequestExists,
			want: codes.AlreadyExists,
		},
		{
			name: "friend already exists",
			err:  statecontract.ErrFriendAlreadyExists,
			want: codes.AlreadyExists,
		},
		{
			name: "invalid friend request",
			err:  statecontract.ErrInvalidFriendRequest,
			want: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(&fakeStateClient{err: tt.err})

			_, err := server.SendFriendRequest(context.Background(), &statepb.SendFriendRequestRequest{
				FromPlayerId: 7,
				ToPlayerId:   8,
			})
			if status.Code(err) != tt.want {
				t.Fatalf("SendFriendRequest code = %v, want %v", status.Code(err), tt.want)
			}
		})
	}
}

func newTestServer(state *fakeStateClient) *Server {
	return NewServer(ServerConfig{
		StateClient:       state,
		PresenceClient:    state,
		FriendClient:      state,
		CoinClient:        state,
		ChatClient:        state,
		LeaderboardClient: state,
	})
}

type fakeStateClient struct {
	account                            *statecontract.Account
	createdAccount                     *statecontract.Account
	session                            *statecontract.Session
	createdSession                     *statecontract.Session
	deletedToken                       string
	player                             *statecontract.Player
	createdPlayer                      *statecontract.Player
	presence                           *statecontract.Presence
	setPresence                        *statecontract.Presence
	setPresenceTTL                     time.Duration
	clearedPlayerID                    int64
	clearedServerName                  string
	refreshedPlayerID                  int64
	refreshedServerName                string
	refreshedAt                        time.Time
	refreshedTTL                       time.Duration
	registerInput                      statecontract.RegisterAccountInput
	registerResult                     *statecontract.RegisterAccountResult
	nextPlayerID                       int64
	err                                error
	gotUsername                        string
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
	settleMatchRewardsInput            statecontract.SettleMatchRewardsInput
	settleMatchRewardsResult           *statecontract.SettleMatchRewardsResult
	saveChatMessageInput               statecontract.SaveChatMessageInput
	savedChatMessage                   *statecontract.ChatMessage
	listChatMessagesInput              statecontract.ListChatMessagesInput
	chatMessages                       []*statecontract.ChatMessage
	listLeaderboardInput               statecontract.ListLeaderboardInput
	listLeaderboardResult              *statecontract.ListLeaderboardResult
	updatedPlayerAvatarID              int64
	updatedAvatar                      string
}

func (f *fakeStateClient) UpdatePlayerAvatar(_ context.Context, playerID int64, avatar string) (*statecontract.Player, error) {
	f.updatedPlayerAvatarID = playerID
	f.updatedAvatar = avatar
	if f.err != nil {
		return nil, f.err
	}
	player := &statecontract.Player{ID: playerID, Avatar: avatar}
	if f.player != nil {
		player = new(*f.player)
		player.Avatar = avatar
	}
	f.player = player
	return new(*player), nil
}

func (f *fakeStateClient) CreateAccount(_ context.Context, account *statecontract.Account) error {
	f.createdAccount = account
	return f.err
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
	if f.err != nil {
		return nil, f.err
	}
	return f.registerResult, nil
}

func (f *fakeStateClient) CreateSession(_ context.Context, session *statecontract.Session) error {
	f.createdSession = session
	return f.err
}

func (f *fakeStateClient) GetSession(_ context.Context, _ string) (*statecontract.Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

func (f *fakeStateClient) DeleteSession(_ context.Context, token string) error {
	f.deletedToken = token
	return f.err
}

func (f *fakeStateClient) CreatePlayer(_ context.Context, player *statecontract.Player) error {
	f.createdPlayer = player
	return f.err
}

func (f *fakeStateClient) GetPlayer(_ context.Context, _ int64) (*statecontract.Player, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.player, nil
}

func (f *fakeStateClient) NextPlayerID(_ context.Context) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.nextPlayerID, nil
}

func (f *fakeStateClient) SettleMatchRewards(_ context.Context, input statecontract.SettleMatchRewardsInput) (*statecontract.SettleMatchRewardsResult, error) {
	f.settleMatchRewardsInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.settleMatchRewardsResult, nil
}

func (f *fakeStateClient) SaveChatMessage(_ context.Context, input statecontract.SaveChatMessageInput) (*statecontract.ChatMessage, error) {
	f.saveChatMessageInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.savedChatMessage, nil
}

func (f *fakeStateClient) ListChatMessages(_ context.Context, input statecontract.ListChatMessagesInput) ([]*statecontract.ChatMessage, error) {
	f.listChatMessagesInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.chatMessages, nil
}

func (f *fakeStateClient) ListLeaderboard(_ context.Context, input statecontract.ListLeaderboardInput) (*statecontract.ListLeaderboardResult, error) {
	f.listLeaderboardInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.listLeaderboardResult, nil
}

func (f *fakeStateClient) SetPresence(_ context.Context, presence *statecontract.Presence, ttl time.Duration) error {
	f.setPresence = presence
	f.setPresenceTTL = ttl
	return f.err
}

func (f *fakeStateClient) GetPresence(_ context.Context, _ int64) (*statecontract.Presence, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.presence, nil
}

func (f *fakeStateClient) ClearPresence(_ context.Context, playerID int64, serverName string) error {
	f.clearedPlayerID = playerID
	f.clearedServerName = serverName
	return f.err
}

func (f *fakeStateClient) RefreshPresence(_ context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	f.refreshedPlayerID = playerID
	f.refreshedServerName = serverName
	f.refreshedAt = updatedAt
	f.refreshedTTL = ttl
	return f.err
}

func (f *fakeStateClient) SendFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.sentFriendRequestFrom = fromPlayerID
	f.sentFriendRequestTo = toPlayerID
	return f.err
}

func (f *fakeStateClient) ListIncomingFriendRequests(_ context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	f.listIncomingFriendRequestsPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.incomingFriendRequests, nil
}

func (f *fakeStateClient) ListOutgoingFriendRequests(_ context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	f.listOutgoingFriendRequestsPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.outgoingFriendRequests, nil
}

func (f *fakeStateClient) AcceptFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.acceptedFriendRequestFrom = fromPlayerID
	f.acceptedFriendRequestTo = toPlayerID
	return f.err
}

func (f *fakeStateClient) RejectFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.rejectedFriendRequestFrom = fromPlayerID
	f.rejectedFriendRequestTo = toPlayerID
	return f.err
}

func (f *fakeStateClient) ListFriendIDs(_ context.Context, playerID int64) ([]int64, error) {
	f.listFriendIDsPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.friendIDs, nil
}

func (f *fakeStateClient) DeleteFriend(_ context.Context, playerID, friendPlayerID int64) error {
	f.deletedFriendPlayerID = playerID
	f.deletedFriendFriendPlayerID = friendPlayerID
	return f.err
}

var _ statecontract.Client = (*fakeStateClient)(nil)
var _ statecontract.PresenceClient = (*fakeStateClient)(nil)
var _ statecontract.FriendClient = (*fakeStateClient)(nil)
var _ statecontract.CoinClient = (*fakeStateClient)(nil)
var _ statecontract.ChatClient = (*fakeStateClient)(nil)
