package realtime

import (
	"testing"
	"time"

	"server/internal/contract/realtimepb"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/leaderboard"
	"server/internal/logic/player"
	"server/internal/logic/presence"
)

func TestToProtoPlayer(t *testing.T) {
	got := toProtoPlayer(&player.Player{ID: 7, Nickname: "Alice", Avatar: "avatar", Email: "a@example.com", Phone: "123", Coins: 99})
	if got.GetId() != 7 || got.GetNickname() != "Alice" || got.GetAvatar() != "avatar" || got.GetEmail() != "a@example.com" || got.GetPhone() != "123" || got.GetCoins() != 99 {
		t.Fatalf("toProtoPlayer() = %+v, want all player fields", got)
	}
	if got := toProtoPlayer(nil); got != nil {
		t.Fatalf("toProtoPlayer(nil) = %+v, want nil", got)
	}
}

func TestToProtoGrowthAndUpgradeResult(t *testing.T) {
	value := &growth.Growth{PlayerID: 7, AttackLevel: 2, AttackSpeedLevel: 3, HealthLevel: 4, MoveSpeedLevel: 5}
	options := []growth.UpgradeOption{{Type: growth.UpgradeAttack, CurrentLevel: 2, NextCost: 100, MaxLevel: 10}}

	got := toProtoGrowth(value, options)
	if got.GetPlayerId() != 7 || got.GetAttackLevel() != 2 || got.GetAttackSpeedLevel() != 3 || got.GetHealthLevel() != 4 || got.GetMoveSpeedLevel() != 5 {
		t.Fatalf("toProtoGrowth() = %+v, want all growth fields", got)
	}
	if len(got.GetUpgradeOptions()) != 1 || got.GetUpgradeOptions()[0].GetType() != "attack" || got.GetUpgradeOptions()[0].GetCurrentLevel() != 2 || got.GetUpgradeOptions()[0].GetNextCost() != 100 || got.GetUpgradeOptions()[0].GetMaxLevel() != 10 {
		t.Fatalf("toProtoGrowth() options = %+v, want converted option", got.GetUpgradeOptions())
	}

	result := toProtoUpgradeGrowthResponse(&growth.UpgradeResult{Growth: value, RemainingCoins: 900, Cost: 100}, options)
	if result.GetRemainingCoins() != 900 || result.GetCost() != 100 || result.GetGrowth().GetPlayerId() != 7 {
		t.Fatalf("toProtoUpgradeGrowthResponse() = %+v, want converted upgrade result", result)
	}
	if got := toProtoGrowth(nil, options); got != nil {
		t.Fatalf("toProtoGrowth(nil, options) = %+v, want nil", got)
	}
}

func TestToProtoFriendModels(t *testing.T) {
	createdAt := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	request := &friend.Request{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: createdAt}
	gotRequest := toProtoFriendRequest(request)
	if gotRequest.GetFromPlayerId() != 7 || gotRequest.GetToPlayerId() != 8 || !gotRequest.GetCreatedAt().AsTime().Equal(createdAt) {
		t.Fatalf("toProtoFriendRequest() = %+v, want converted request", gotRequest)
	}
	requests := toProtoFriendRequests([]*friend.Request{request, nil})
	if len(requests) != 1 || requests[0].GetFromPlayerId() != 7 {
		t.Fatalf("toProtoFriendRequests() = %+v, want one non-nil request", requests)
	}

	updatedAt := createdAt.Add(time.Minute)
	summary := toProtoFriendSummary(&player.Player{ID: 8, Nickname: "Bob", Avatar: "avatar"}, &presence.Presence{Status: presence.StatusOnline, UpdatedAt: updatedAt})
	if summary.GetPlayerId() != 8 || summary.GetNickname() != "Bob" || summary.GetAvatar() != "avatar" || !summary.GetOnline() || summary.GetStatus() != presence.StatusOnline || !summary.GetUpdatedAt().AsTime().Equal(updatedAt) {
		t.Fatalf("toProtoFriendSummary() = %+v, want converted online friend", summary)
	}
	if got := toProtoFriendSummary(nil, nil); got != nil {
		t.Fatalf("toProtoFriendSummary(nil, nil) = %+v, want nil", got)
	}
}

func TestLeaderboardTypeConversions(t *testing.T) {
	tests := []struct {
		name       string
		protoType  realtimepb.LeaderboardType
		domainType leaderboard.Type
	}{
		{
			name:       "solo",
			protoType:  realtimepb.LeaderboardType_LEADERBOARD_TYPE_SOLO_CLEAR_TIME,
			domainType: leaderboard.TypeSoloClearTime,
		},
		{
			name:       "duo",
			protoType:  realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME,
			domainType: leaderboard.TypeDuoClearTime,
		},
		{
			name:       "total kills",
			protoType:  realtimepb.LeaderboardType_LEADERBOARD_TYPE_TOTAL_KILLS,
			domainType: leaderboard.TypeTotalKills,
		},
		{
			name:       "unknown",
			protoType:  realtimepb.LeaderboardType(99),
			domainType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toLeaderboardType(tt.protoType); got != tt.domainType {
				t.Fatalf("toLeaderboardType(%v) = %q, want %q", tt.protoType, got, tt.domainType)
			}
			wantProtoType := tt.protoType
			if tt.domainType == "" {
				wantProtoType = realtimepb.LeaderboardType_LEADERBOARD_TYPE_UNSPECIFIED
			}
			if got := toProtoLeaderboardType(tt.domainType); got != wantProtoType {
				t.Fatalf("toProtoLeaderboardType(%q) = %v, want %v", tt.domainType, got, wantProtoType)
			}
		})
	}
}

func TestLeaderboardProtocolConversions(t *testing.T) {
	input := toLeaderboardListInput(&realtimepb.ListLeaderboardRequest{
		Type:       realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME,
		MapVersion: "wave-v1",
		Limit:      10,
	})
	if input.Type != leaderboard.TypeDuoClearTime || input.MapVersion != "wave-v1" || input.Limit != 10 {
		t.Fatalf("toLeaderboardListInput() = %+v, want duo wave-v1 limit 10", input)
	}

	response := toProtoLeaderboardResult(&leaderboard.Result{
		Type:       leaderboard.TypeDuoClearTime,
		MapVersion: "wave-v1",
		Entries: []leaderboard.Entry{
			{
				Rank:  1,
				Score: 82_000,
				Players: []leaderboard.Player{
					{PlayerID: 7, Nickname: "Alice", Avatar: "mage"},
					{PlayerID: 8, Nickname: "Bob", Avatar: "knight"},
				},
			},
		},
	})
	if response.GetType() != realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME || response.GetMapVersion() != "wave-v1" || len(response.GetEntries()) != 1 {
		t.Fatalf("toProtoLeaderboardResult() = %+v, want converted metadata and entry", response)
	}
	entry := response.GetEntries()[0]
	if entry.GetRank() != 1 || entry.GetScore() != 82_000 || len(entry.GetPlayers()) != 2 {
		t.Fatalf("leaderboard entry = %+v, want rank 1 score 82000 and two players", entry)
	}
	if entry.GetPlayers()[0].GetNickname() != "Alice" || entry.GetPlayers()[0].GetAvatar() != "mage" || entry.GetPlayers()[1].GetPlayerId() != 8 {
		t.Fatalf("leaderboard players = %+v, want complete converted players", entry.GetPlayers())
	}
}

func TestLeaderboardProtocolConversionsHandleNil(t *testing.T) {
	if got := toLeaderboardListInput(nil); got != (leaderboard.ListInput{}) {
		t.Fatalf("toLeaderboardListInput(nil) = %+v, want empty", got)
	}
	if got := toProtoLeaderboardResult(nil); got != nil {
		t.Fatalf("toProtoLeaderboardResult(nil) = %+v, want nil", got)
	}
}
