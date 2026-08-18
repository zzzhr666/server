package stateproto

import (
	"testing"
	"time"

	statecontract "server/internal/contract/state"
	"server/internal/contract/statepb"
)

func TestMatchLeaderboardRecordRoundTrip(t *testing.T) {
	want := &statecontract.MatchLeaderboardRecord{
		Mode:             "duo",
		MapVersion:       "wave-v1",
		Cleared:          true,
		CombatDurationMS: 83_250,
		Players: []statecontract.PlayerLeaderboardRecord{
			{PlayerID: 8, TotalKills: 17},
			{PlayerID: 7, TotalKills: 13},
		},
	}

	got := FromProtoMatchLeaderboardRecord(ToProtoMatchLeaderboardRecord(want))
	if got == nil {
		t.Fatal("leaderboard round trip = nil, want record")
	}
	if got.Mode != want.Mode || got.MapVersion != want.MapVersion || got.Cleared != want.Cleared || got.CombatDurationMS != want.CombatDurationMS {
		t.Fatalf("leaderboard round trip = %+v, want metadata %+v", got, want)
	}
	if len(got.Players) != 2 || got.Players[0] != want.Players[0] || got.Players[1] != want.Players[1] {
		t.Fatalf("leaderboard players = %+v, want %+v", got.Players, want.Players)
	}
}

func TestLeaderboardConvertersHandleNil(t *testing.T) {
	if ToProtoPlayerLeaderboardRecord(nil) != nil {
		t.Fatal("ToProtoPlayerLeaderboardRecord(nil) != nil")
	}
	if FromProtoPlayerLeaderboardRecord(nil) != nil {
		t.Fatal("FromProtoPlayerLeaderboardRecord(nil) != nil")
	}
	if ToProtoMatchLeaderboardRecord(nil) != nil {
		t.Fatal("ToProtoMatchLeaderboardRecord(nil) != nil")
	}
	if FromProtoMatchLeaderboardRecord(nil) != nil {
		t.Fatal("FromProtoMatchLeaderboardRecord(nil) != nil")
	}
}

func TestFromProtoMatchLeaderboardRecordSkipsNilPlayers(t *testing.T) {
	got := FromProtoMatchLeaderboardRecord(&statepb.MatchLeaderboardRecord{
		Mode:       "solo",
		MapVersion: "wave-v1",
		Players: []*statepb.PlayerLeaderboardRecord{
			nil,
			{PlayerId: 7, TotalKills: 19},
		},
	})

	if len(got.Players) != 1 || got.Players[0].PlayerID != 7 || got.Players[0].TotalKills != 19 {
		t.Fatalf("leaderboard players = %+v, want one converted non-nil player", got.Players)
	}
}

func TestLeaderboardResultRoundTrip(t *testing.T) {
	want := &statecontract.ListLeaderboardResult{
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
	}

	got := FromProtoLeaderboardResult(ToProtoLeaderboardResult(want))
	if got == nil || got.Type != want.Type || got.MapVersion != want.MapVersion || len(got.Entries) != 1 {
		t.Fatalf("leaderboard round trip = %+v, want metadata %+v", got, want)
	}
	entry := got.Entries[0]
	if entry.Rank != 1 || entry.Score != 82_000 || len(entry.Players) != 2 {
		t.Fatalf("leaderboard entry = %+v, want complete entry", entry)
	}
	if entry.Players[0] != want.Entries[0].Players[0] || entry.Players[1] != want.Entries[0].Players[1] {
		t.Fatalf("leaderboard players = %+v, want %+v", entry.Players, want.Entries[0].Players)
	}
}

func TestLeaderboardTypeConversions(t *testing.T) {
	tests := []struct {
		name       string
		domainType statecontract.LeaderboardType
		protoType  statepb.LeaderboardType
	}{
		{name: "solo", domainType: statecontract.LeaderboardTypeSoloClearTime, protoType: statepb.LeaderboardType_SOLO_CLEAR_TIME},
		{name: "duo", domainType: statecontract.LeaderboardTypeDuoClearTime, protoType: statepb.LeaderboardType_DUO_CLEAR_TIME},
		{name: "total kills", domainType: statecontract.LeaderboardTypeTotalKills, protoType: statepb.LeaderboardType_TOTAL_KILLS},
		{name: "unknown", domainType: "unknown", protoType: statepb.LeaderboardType_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToProtoLeaderboardType(tt.domainType); got != tt.protoType {
				t.Fatalf("ToProtoLeaderboardType(%q) = %v, want %v", tt.domainType, got, tt.protoType)
			}
			wantDomainType := tt.domainType
			if tt.protoType == statepb.LeaderboardType_UNSPECIFIED {
				wantDomainType = ""
			}
			if got := FromProtoLeaderboardType(tt.protoType); got != wantDomainType {
				t.Fatalf("FromProtoLeaderboardType(%v) = %q, want %q", tt.protoType, got, wantDomainType)
			}
		})
	}
}

func TestLeaderboardResultConvertersHandleNilAndSkipNilMessages(t *testing.T) {
	if ToProtoLeaderboardResult(nil) != nil {
		t.Fatal("ToProtoLeaderboardResult(nil) != nil")
	}
	if FromProtoLeaderboardResult(nil) != nil {
		t.Fatal("FromProtoLeaderboardResult(nil) != nil")
	}

	got := FromProtoLeaderboardResult(&statepb.ListLeaderboardResponse{
		Type: statepb.LeaderboardType_SOLO_CLEAR_TIME,
		Entries: []*statepb.LeaderboardEntry{
			nil,
			{
				Rank: 1,
				Players: []*statepb.LeaderboardPlayer{
					nil,
					{PlayerId: 7, Nickname: "Alice", Avatar: "mage"},
				},
				Score: 80_000,
			},
		},
	})
	if len(got.Entries) != 1 || len(got.Entries[0].Players) != 1 || got.Entries[0].Players[0].PlayerID != 7 {
		t.Fatalf("converted leaderboard = %+v, want nil messages skipped", got)
	}
}

func TestRealtimeDeliveryRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	want := &statecontract.RealtimeDelivery{
		Route: statecontract.RealtimeRoute{
			Type:       statecontract.RealtimeRouteServer,
			ServerName: "logic-2",
		},
		Event: &statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventChatMessage,
			TargetPlayerID: 8,
			ActorPlayerID:  7,
			ChatMessage: &statecontract.ChatMessage{
				MessageKey:       "message-1",
				ChannelType:      statecontract.ChatChannelDirect,
				ChannelKey:       "direct:7:8",
				SenderID:         7,
				ReceiverID:       8,
				SenderNickname:   "Alice",
				Content:          "hello",
				CreatedAt:        createdAt,
				ExpiresAt:        createdAt.Add(statecontract.DirectChatRetention),
				ClientMessageKey: "client-1",
			},
		},
	}

	got := FromProtoRealtimeDelivery(ToProtoRealtimeDelivery(want))
	if got == nil || got.Event == nil || got.Event.ChatMessage == nil {
		t.Fatalf("delivery round trip = %+v, want complete delivery", got)
	}
	if got.Route != want.Route || got.Event.Type != want.Event.Type || got.Event.TargetPlayerID != want.Event.TargetPlayerID || got.Event.ActorPlayerID != want.Event.ActorPlayerID {
		t.Fatalf("delivery round trip = %+v, want route and event %+v", got, want)
	}
	message := got.Event.ChatMessage
	wantMessage := want.Event.ChatMessage
	if message.MessageKey != wantMessage.MessageKey || message.ChannelType != wantMessage.ChannelType || message.ChannelKey != wantMessage.ChannelKey || message.SenderID != wantMessage.SenderID || message.ReceiverID != wantMessage.ReceiverID || message.SenderNickname != wantMessage.SenderNickname || message.Content != wantMessage.Content || message.ClientMessageKey != wantMessage.ClientMessageKey || !message.CreatedAt.Equal(wantMessage.CreatedAt) || !message.ExpiresAt.Equal(wantMessage.ExpiresAt) {
		t.Fatalf("chat message round trip = %+v, want %+v", message, wantMessage)
	}
}

func TestRealtimeDeliveryConvertersHandleNil(t *testing.T) {
	if ToProtoRealtimeDelivery(nil) != nil {
		t.Fatal("ToProtoRealtimeDelivery(nil) != nil")
	}
	if FromProtoRealtimeDelivery(nil) != nil {
		t.Fatal("FromProtoRealtimeDelivery(nil) != nil")
	}
}
