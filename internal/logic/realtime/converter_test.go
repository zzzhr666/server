package realtime

import (
	"testing"
	"time"

	"server/internal/logic/friend"
	"server/internal/logic/growth"
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
