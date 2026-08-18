package redisstore

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"reflect"
	statecontract "server/internal/contract/state"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestResolveLeaderboardQuery(t *testing.T) {
	tests := []struct {
		name           string
		input          statecontract.ListLeaderboardInput
		wantKey        string
		wantDescending bool
		wantErr        error
	}{
		{
			name: "solo clear time",
			input: statecontract.ListLeaderboardInput{
				Type:       statecontract.LeaderboardTypeSoloClearTime,
				MapVersion: "wave-v1",
				Limit:      10,
			},
			wantKey: clearTimeLeaderboardKey(leaderboardModeSolo, "wave-v1"),
		},
		{
			name: "duo clear time",
			input: statecontract.ListLeaderboardInput{
				Type:       statecontract.LeaderboardTypeDuoClearTime,
				MapVersion: "rooms-v1",
				Limit:      maxLeaderboardLimit,
			},
			wantKey: clearTimeLeaderboardKey(leaderboardModeDuo, "rooms-v1"),
		},
		{
			name: "total kills ignores map version",
			input: statecontract.ListLeaderboardInput{
				Type:       statecontract.LeaderboardTypeTotalKills,
				MapVersion: "wave-v1",
				Limit:      10,
			},
			wantKey:        totalKillsLeaderboardKey,
			wantDescending: true,
		},
		{
			name: "zero limit",
			input: statecontract.ListLeaderboardInput{
				Type:       statecontract.LeaderboardTypeSoloClearTime,
				MapVersion: "wave-v1",
			},
			wantErr: statecontract.ErrInvalidLeaderboardQuery,
		},
		{
			name: "negative limit",
			input: statecontract.ListLeaderboardInput{
				Type:       statecontract.LeaderboardTypeSoloClearTime,
				MapVersion: "wave-v1",
				Limit:      -1,
			},
			wantErr: statecontract.ErrInvalidLeaderboardQuery,
		},
		{
			name: "limit exceeds maximum",
			input: statecontract.ListLeaderboardInput{
				Type:       statecontract.LeaderboardTypeSoloClearTime,
				MapVersion: "wave-v1",
				Limit:      maxLeaderboardLimit + 1,
			},
			wantErr: statecontract.ErrInvalidLeaderboardQuery,
		},
		{
			name: "clear time without map version",
			input: statecontract.ListLeaderboardInput{
				Type:  statecontract.LeaderboardTypeSoloClearTime,
				Limit: 10,
			},
			wantErr: statecontract.ErrInvalidLeaderboardQuery,
		},
		{
			name: "unknown type",
			input: statecontract.ListLeaderboardInput{
				Type:  "unknown",
				Limit: 10,
			},
			wantErr: statecontract.ErrInvalidLeaderboardQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, descending, err := resolveLeaderboardQuery(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveLeaderboardQuery error = %v, want %v", err, tt.wantErr)
			}
			if key != tt.wantKey || descending != tt.wantDescending {
				t.Fatalf("resolveLeaderboardQuery = (%q, %v), want (%q, %v)", key, descending, tt.wantKey, tt.wantDescending)
			}
		})
	}
}

func TestParseLeaderboardMember(t *testing.T) {
	tests := []struct {
		name      string
		boardType statecontract.LeaderboardType
		member    string
		want      []int64
		queryErr  bool
		memberErr bool
	}{
		{name: "solo", boardType: statecontract.LeaderboardTypeSoloClearTime, member: "7", want: []int64{7}},
		{name: "total kills", boardType: statecontract.LeaderboardTypeTotalKills, member: "8", want: []int64{8}},
		{name: "duo", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "7:8", want: []int64{7, 8}},
		{name: "duo missing player", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "7", memberErr: true},
		{name: "duo extra player", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "7:8:9", memberErr: true},
		{name: "duo nonnumeric", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "7:eight", memberErr: true},
		{name: "duo nonpositive", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "0:8", memberErr: true},
		{name: "duo duplicate", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "7:7", memberErr: true},
		{name: "duo reversed", boardType: statecontract.LeaderboardTypeDuoClearTime, member: "8:7", memberErr: true},
		{name: "solo nonnumeric", boardType: statecontract.LeaderboardTypeSoloClearTime, member: "seven", memberErr: true},
		{name: "solo nonpositive", boardType: statecontract.LeaderboardTypeSoloClearTime, member: "0", memberErr: true},
		{name: "unknown type", boardType: "unknown", member: "7", queryErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLeaderboardMember(tt.boardType, tt.member)
			switch {
			case tt.queryErr && !errors.Is(err, statecontract.ErrInvalidLeaderboardQuery):
				t.Fatalf("parseLeaderboardMember error = %v, want %v", err, statecontract.ErrInvalidLeaderboardQuery)
			case tt.memberErr && (err == nil || errors.Is(err, statecontract.ErrInvalidLeaderboardQuery)):
				t.Fatalf("parseLeaderboardMember error = %v, want stored member error", err)
			case !tt.queryErr && !tt.memberErr && err != nil:
				t.Fatalf("parseLeaderboardMember returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseLeaderboardMember = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListLeaderboardOrdersAndLimitsSoloClearTimes(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	key := clearTimeLeaderboardKey(leaderboardModeSolo, "wave-v1")
	if err := client.ZAdd(ctx, key,
		redis.Z{Member: "7", Score: 90_000},
		redis.Z{Member: "8", Score: 80_000},
		redis.Z{Member: "9", Score: 100_000},
	).Err(); err != nil {
		t.Fatalf("seed solo leaderboard returned error: %v", err)
	}

	result, err := store.ListLeaderboard(ctx, statecontract.ListLeaderboardInput{
		Type:       statecontract.LeaderboardTypeSoloClearTime,
		MapVersion: "wave-v1",
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("ListLeaderboard returned error: %v", err)
	}
	if result.Type != statecontract.LeaderboardTypeSoloClearTime || result.MapVersion != "wave-v1" {
		t.Fatalf("leaderboard metadata = (%q, %q), want (%q, %q)", result.Type, result.MapVersion, statecontract.LeaderboardTypeSoloClearTime, "wave-v1")
	}
	assertLeaderboardEntries(t, result.Entries, []leaderboardEntryWant{
		{rank: 1, score: 80_000, playerIDs: []int64{8}},
		{rank: 2, score: 90_000, playerIDs: []int64{7}},
	})
}

func TestListLeaderboardExpandsCanonicalDuoMembers(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	if err := client.HSet(ctx, playerKey(7), map[string]any{
		"nickname": "player-seven",
		"avatar":   "avatar-seven",
	}).Err(); err != nil {
		t.Fatalf("seed player profile returned error: %v", err)
	}
	if err := client.HSet(ctx, playerKey(9), map[string]any{
		"nickname": "player-nine",
		"avatar":   "avatar-nine",
	}).Err(); err != nil {
		t.Fatalf("seed player profile returned error: %v", err)
	}
	key := clearTimeLeaderboardKey(leaderboardModeDuo, "wave-v1")
	if err := client.ZAdd(ctx, key,
		redis.Z{Member: "7:8", Score: 93_000},
		redis.Z{Member: "7:9", Score: 82_000},
	).Err(); err != nil {
		t.Fatalf("seed duo leaderboard returned error: %v", err)
	}

	result, err := store.ListLeaderboard(ctx, statecontract.ListLeaderboardInput{
		Type:       statecontract.LeaderboardTypeDuoClearTime,
		MapVersion: "wave-v1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("ListLeaderboard returned error: %v", err)
	}
	assertLeaderboardEntries(t, result.Entries, []leaderboardEntryWant{
		{
			rank:  1,
			score: 82_000,
			players: []leaderboardPlayerWant{
				{playerID: 7, nickname: "player-seven", avatar: "avatar-seven"},
				{playerID: 9, nickname: "player-nine", avatar: "avatar-nine"},
			},
		},
		{
			rank:  2,
			score: 93_000,
			players: []leaderboardPlayerWant{
				{playerID: 7, nickname: "player-seven", avatar: "avatar-seven"},
				{playerID: 8},
			},
		},
	})
}

func TestListLeaderboardOrdersTotalKillsDescending(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	if err := client.ZAdd(ctx, totalKillsLeaderboardKey,
		redis.Z{Member: "7", Score: 50},
		redis.Z{Member: "8", Score: 100},
		redis.Z{Member: "9", Score: 75},
	).Err(); err != nil {
		t.Fatalf("seed total kills leaderboard returned error: %v", err)
	}

	result, err := store.ListLeaderboard(ctx, statecontract.ListLeaderboardInput{
		Type:       statecontract.LeaderboardTypeTotalKills,
		MapVersion: "ignored",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("ListLeaderboard returned error: %v", err)
	}
	if result.MapVersion != "" {
		t.Fatalf("MapVersion = %q, want empty", result.MapVersion)
	}
	assertLeaderboardEntries(t, result.Entries, []leaderboardEntryWant{
		{rank: 1, score: 100, playerIDs: []int64{8}},
		{rank: 2, score: 75, playerIDs: []int64{9}},
		{rank: 3, score: 50, playerIDs: []int64{7}},
	})
}

func TestListLeaderboardReturnsEmptyEntriesForMissingBoard(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisTestStore(t)

	result, err := store.ListLeaderboard(ctx, statecontract.ListLeaderboardInput{
		Type:       statecontract.LeaderboardTypeSoloClearTime,
		MapVersion: "missing",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("ListLeaderboard returned error: %v", err)
	}
	if result.Entries == nil || len(result.Entries) != 0 {
		t.Fatalf("Entries = %#v, want non-nil empty slice", result.Entries)
	}
}

type leaderboardEntryWant struct {
	rank      int64
	score     int64
	playerIDs []int64
	players   []leaderboardPlayerWant
}

type leaderboardPlayerWant struct {
	playerID int64
	nickname string
	avatar   string
}

func assertLeaderboardEntries(t *testing.T, got []statecontract.LeaderboardEntry, want []leaderboardEntryWant) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("entries count = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Rank != want[i].rank || got[i].Score != want[i].score {
			t.Fatalf("entry %d = %+v, want rank %d score %d", i, got[i], want[i].rank, want[i].score)
		}
		wantPlayers := want[i].players
		if wantPlayers == nil {
			wantPlayers = make([]leaderboardPlayerWant, len(want[i].playerIDs))
			for j, playerID := range want[i].playerIDs {
				wantPlayers[j].playerID = playerID
			}
		}
		if len(got[i].Players) != len(wantPlayers) {
			t.Fatalf("entry %d players = %+v, want %+v", i, got[i].Players, wantPlayers)
		}
		for j, wantPlayer := range wantPlayers {
			gotPlayer := got[i].Players[j]
			if gotPlayer.PlayerID != wantPlayer.playerID || gotPlayer.Nickname != wantPlayer.nickname || gotPlayer.Avatar != wantPlayer.avatar {
				t.Fatalf("entry %d player %d = %+v, want %+v", i, j, gotPlayer, wantPlayer)
			}
		}
	}
}

func TestFriendRequestLifecycleAccept(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayer(t, ctx, client, 7)
	seedPlayer(t, ctx, client, 8)

	if err := store.SendFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}

	requestValues, err := client.HGetAll(ctx, friendRequestKey(7, 8)).Result()
	if err != nil {
		t.Fatalf("HGetAll friend request returned error: %v", err)
	}
	if requestValues["from_player_id"] != "7" {
		t.Fatalf("from_player_id = %q, want 7", requestValues["from_player_id"])
	}
	if requestValues["to_player_id"] != "8" {
		t.Fatalf("to_player_id = %q, want 8", requestValues["to_player_id"])
	}
	if requestValues["created_at"] == "" {
		t.Fatalf("created_at is empty")
	}

	assertZSetMembers(t, ctx, client, friendIncomingKey(8), []string{"7"})
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), []string{"8"})

	incoming, err := store.ListIncomingFriendRequests(ctx, 8)
	if err != nil {
		t.Fatalf("ListIncomingFriendRequests returned error: %v", err)
	}
	assertFriendRequests(t, incoming, []friendRequestWant{{from: 7, to: 8}})

	outgoing, err := store.ListOutgoingFriendRequests(ctx, 7)
	if err != nil {
		t.Fatalf("ListOutgoingFriendRequests returned error: %v", err)
	}
	assertFriendRequests(t, outgoing, []friendRequestWant{{from: 7, to: 8}})

	if err := store.SendFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendRequestExists) {
		t.Fatalf("duplicate SendFriendRequest error = %v, want %v", err, statecontract.ErrFriendRequestExists)
	}
	if err := store.SendFriendRequest(ctx, 8, 7); !errors.Is(err, statecontract.ErrFriendRequestExists) {
		t.Fatalf("reverse SendFriendRequest error = %v, want %v", err, statecontract.ErrFriendRequestExists)
	}

	if err := store.AcceptFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("AcceptFriendRequest returned error: %v", err)
	}

	assertSetMembers(t, ctx, client, friendsKey(7), []string{"8"})
	assertSetMembers(t, ctx, client, friendsKey(8), []string{"7"})
	assertHashMissing(t, ctx, client, friendRequestKey(7, 8))
	assertZSetMembers(t, ctx, client, friendIncomingKey(8), nil)
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), nil)

	friendIDs, err := store.ListFriendIDs(ctx, 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	assertInt64Set(t, friendIDs, []int64{8})

	if err := store.SendFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendAlreadyExists) {
		t.Fatalf("SendFriendRequest for existing friends error = %v, want %v", err, statecontract.ErrFriendAlreadyExists)
	}
}

func TestFriendRequestLifecycleReject(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayer(t, ctx, client, 7)
	seedPlayer(t, ctx, client, 8)

	if err := store.SendFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}
	if err := store.RejectFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("RejectFriendRequest returned error: %v", err)
	}

	assertHashMissing(t, ctx, client, friendRequestKey(7, 8))
	assertZSetMembers(t, ctx, client, friendIncomingKey(8), nil)
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), nil)
	assertSetMembers(t, ctx, client, friendsKey(7), nil)
	assertSetMembers(t, ctx, client, friendsKey(8), nil)

	if err := store.RejectFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendRequestNotFound) {
		t.Fatalf("second RejectFriendRequest error = %v, want %v", err, statecontract.ErrFriendRequestNotFound)
	}
}

func TestDeleteFriendRemovesBothDirections(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayer(t, ctx, client, 7)
	seedPlayer(t, ctx, client, 8)

	if err := store.SendFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}
	if err := store.AcceptFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("AcceptFriendRequest returned error: %v", err)
	}

	if err := store.DeleteFriend(ctx, 7, 8); err != nil {
		t.Fatalf("DeleteFriend returned error: %v", err)
	}

	assertSetMembers(t, ctx, client, friendsKey(7), nil)
	assertSetMembers(t, ctx, client, friendsKey(8), nil)

	friendIDs, err := store.ListFriendIDs(ctx, 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	assertInt64Set(t, friendIDs, nil)

	if err := store.DeleteFriend(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendNotFound) {
		t.Fatalf("second DeleteFriend error = %v, want %v", err, statecontract.ErrFriendNotFound)
	}
}

func TestSendFriendRequestRequiresExistingPlayers(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayer(t, ctx, client, 7)

	if err := store.SendFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrPlayerNotFound) {
		t.Fatalf("SendFriendRequest error = %v, want %v", err, statecontract.ErrPlayerNotFound)
	}
	assertHashMissing(t, ctx, client, friendRequestKey(7, 8))
	assertZSetMembers(t, ctx, client, friendIncomingKey(8), nil)
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), nil)

	if err := store.SendFriendRequest(ctx, 9, 7); !errors.Is(err, statecontract.ErrPlayerNotFound) {
		t.Fatalf("SendFriendRequest missing sender error = %v, want %v", err, statecontract.ErrPlayerNotFound)
	}
	assertHashMissing(t, ctx, client, friendRequestKey(9, 7))
	assertZSetMembers(t, ctx, client, friendIncomingKey(7), nil)
	assertZSetMembers(t, ctx, client, friendOutgoingKey(9), nil)
}

func TestFriendMethodsValidateInput(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisTestStore(t)

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "send self",
			run: func() error {
				return store.SendFriendRequest(ctx, 7, 7)
			},
		},
		{
			name: "list incoming zero player",
			run: func() error {
				_, err := store.ListIncomingFriendRequests(ctx, 0)
				return err
			},
		},
		{
			name: "list outgoing zero player",
			run: func() error {
				_, err := store.ListOutgoingFriendRequests(ctx, 0)
				return err
			},
		},
		{
			name: "accept self",
			run: func() error {
				return store.AcceptFriendRequest(ctx, 7, 7)
			},
		},
		{
			name: "reject self",
			run: func() error {
				return store.RejectFriendRequest(ctx, 7, 7)
			},
		},
		{
			name: "list friend ids zero player",
			run: func() error {
				_, err := store.ListFriendIDs(ctx, 0)
				return err
			},
		},
		{
			name: "delete self",
			run: func() error {
				return store.DeleteFriend(ctx, 7, 7)
			},
		},
		{
			name: "delete zero friend",
			run: func() error {
				return store.DeleteFriend(ctx, 7, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, statecontract.ErrInvalidFriendRequest) {
				t.Fatalf("error = %v, want %v", err, statecontract.ErrInvalidFriendRequest)
			}
		})
	}
}

func TestSettleMatchRewardsAppliesAtomicallyAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerWithCoins(t, ctx, client, 7, 100)
	seedPlayerWithCoins(t, ctx, client, 8, 200)
	input := statecontract.SettleMatchRewardsInput{
		SettlementID: "room-settlement-1",
		Rewards: []statecontract.PlayerCoinReward{
			{PlayerID: 7, Amount: 50},
			{PlayerID: 8, Amount: 75},
		},
	}

	result, err := store.SettleMatchRewards(ctx, input)
	if err != nil {
		t.Fatalf("SettleMatchRewards returned error: %v", err)
	}
	if result == nil || !result.Applied {
		t.Fatalf("settlement result = %+v, want applied", result)
	}
	assertPlayerCoins(t, ctx, client, 7, 150)
	assertPlayerCoins(t, ctx, client, 8, 275)

	result, err = store.SettleMatchRewards(ctx, input)
	if err != nil {
		t.Fatalf("duplicate SettleMatchRewards returned error: %v", err)
	}
	if result == nil || result.Applied {
		t.Fatalf("duplicate settlement result = %+v, want not applied", result)
	}
	assertPlayerCoins(t, ctx, client, 7, 150)
	assertPlayerCoins(t, ctx, client, 8, 275)
}

func TestSettleMatchRewardsDoesNotPartiallyApplyWhenPlayerMissing(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerWithCoins(t, ctx, client, 7, 100)
	input := statecontract.SettleMatchRewardsInput{
		SettlementID: "room-settlement-2",
		Rewards: []statecontract.PlayerCoinReward{
			{PlayerID: 7, Amount: 50},
			{PlayerID: 8, Amount: 75},
		},
		Leaderboard: &statecontract.MatchLeaderboardRecord{
			Mode:             leaderboardModeDuo,
			MapVersion:       "wave-v1",
			Cleared:          true,
			CombatDurationMS: 83_250,
			Players: []statecontract.PlayerLeaderboardRecord{
				{PlayerID: 8, TotalKills: 3},
				{PlayerID: 7, TotalKills: 5},
			},
		},
	}

	if _, err := store.SettleMatchRewards(ctx, input); !errors.Is(err, statecontract.ErrPlayerNotFound) {
		t.Fatalf("SettleMatchRewards error = %v, want %v", err, statecontract.ErrPlayerNotFound)
	}
	assertPlayerCoins(t, ctx, client, 7, 100)
	assertRedisKeyMissing(t, ctx, client, totalKillsLeaderboardKey)
	assertRedisKeyMissing(t, ctx, client, clearTimeLeaderboardKey(leaderboardModeDuo, "wave-v1"))

	seedPlayerWithCoins(t, ctx, client, 8, 200)
	result, err := store.SettleMatchRewards(ctx, input)
	if err != nil {
		t.Fatalf("retry SettleMatchRewards returned error: %v", err)
	}
	if result == nil || !result.Applied {
		t.Fatalf("retry settlement result = %+v, want applied", result)
	}
	assertPlayerCoins(t, ctx, client, 7, 150)
	assertPlayerCoins(t, ctx, client, 8, 275)
	assertZScore(t, ctx, client, totalKillsLeaderboardKey, "7", 5)
	assertZScore(t, ctx, client, totalKillsLeaderboardKey, "8", 3)
	assertZScore(t, ctx, client, clearTimeLeaderboardKey(leaderboardModeDuo, "wave-v1"), "7:8", 83_250)
}

func TestSettleMatchRewardsConcurrentDuplicateAppliesOnce(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)

	seedPlayerWithCoins(t, ctx, client, 7, 100)
	seedPlayerWithCoins(t, ctx, client, 8, 200)
	input := statecontract.SettleMatchRewardsInput{
		SettlementID: "room-settlement-concurrent",
		Rewards: []statecontract.PlayerCoinReward{
			{PlayerID: 7, Amount: 50},
			{PlayerID: 8, Amount: 75},
		},
		Leaderboard: &statecontract.MatchLeaderboardRecord{
			Mode:             leaderboardModeDuo,
			MapVersion:       "wave-v1",
			Cleared:          true,
			CombatDurationMS: 83_250,
			Players: []statecontract.PlayerLeaderboardRecord{
				{PlayerID: 8, TotalKills: 3},
				{PlayerID: 7, TotalKills: 5},
			},
		},
	}

	const callerCount = 16
	start := make(chan struct{})
	results := make(chan *statecontract.SettleMatchRewardsResult, callerCount)
	errs := make(chan error, callerCount)
	var callers sync.WaitGroup
	for range callerCount {
		callers.Add(1)
		go func() {
			defer callers.Done()
			<-start
			result, err := store.SettleMatchRewards(ctx, input)
			results <- result
			errs <- err
		}()
	}
	close(start)
	callers.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent SettleMatchRewards returned error: %v", err)
		}
	}
	appliedCount := 0
	for result := range results {
		if result == nil {
			t.Fatal("concurrent settlement result = nil")
		}
		if result.Applied {
			appliedCount++
		}
	}
	if appliedCount != 1 {
		t.Fatalf("applied results = %d, want 1", appliedCount)
	}
	assertPlayerCoins(t, ctx, client, 7, 150)
	assertPlayerCoins(t, ctx, client, 8, 275)
	assertZScore(t, ctx, client, totalKillsLeaderboardKey, "7", 5)
	assertZScore(t, ctx, client, totalKillsLeaderboardKey, "8", 3)
	assertZScore(t, ctx, client, clearTimeLeaderboardKey(leaderboardModeDuo, "wave-v1"), "7:8", 83_250)
}

func TestSettleMatchRewardsDefeatOnlyUpdatesTotalKills(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerWithCoins(t, ctx, client, 7, 100)
	input := statecontract.SettleMatchRewardsInput{
		SettlementID: "room-defeat",
		Rewards: []statecontract.PlayerCoinReward{
			{PlayerID: 7, Amount: 50},
		},
		Leaderboard: &statecontract.MatchLeaderboardRecord{
			Mode:             leaderboardModeSolo,
			MapVersion:       "wave-v1",
			CombatDurationMS: 47_000,
			Players: []statecontract.PlayerLeaderboardRecord{
				{PlayerID: 7, TotalKills: 4},
			},
		},
	}

	result, err := store.SettleMatchRewards(ctx, input)
	if err != nil {
		t.Fatalf("SettleMatchRewards returned error: %v", err)
	}
	if result == nil || !result.Applied {
		t.Fatalf("settlement result = %+v, want applied", result)
	}
	assertPlayerCoins(t, ctx, client, 7, 150)
	assertZScore(t, ctx, client, totalKillsLeaderboardKey, "7", 4)
	assertRedisKeyMissing(t, ctx, client, clearTimeLeaderboardKey(leaderboardModeSolo, "wave-v1"))
}

func TestSettleMatchRewardsKeepsShortestSoloClearTime(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerWithCoins(t, ctx, client, 7, 100)

	settle := func(settlementID string, durationMS int64) {
		t.Helper()
		_, err := store.SettleMatchRewards(ctx, statecontract.SettleMatchRewardsInput{
			SettlementID: settlementID,
			Rewards: []statecontract.PlayerCoinReward{
				{PlayerID: 7, Amount: 1},
			},
			Leaderboard: &statecontract.MatchLeaderboardRecord{
				Mode:             leaderboardModeSolo,
				MapVersion:       "wave-v1",
				Cleared:          true,
				CombatDurationMS: durationMS,
				Players: []statecontract.PlayerLeaderboardRecord{
					{PlayerID: 7},
				},
			},
		})
		if err != nil {
			t.Fatalf("SettleMatchRewards duration %d returned error: %v", durationMS, err)
		}
	}

	key := clearTimeLeaderboardKey(leaderboardModeSolo, "wave-v1")
	settle("room-clear-first", 90_000)
	assertZScore(t, ctx, client, key, "7", 90_000)
	settle("room-clear-slower", 100_000)
	assertZScore(t, ctx, client, key, "7", 90_000)
	settle("room-clear-faster", 80_000)
	assertZScore(t, ctx, client, key, "7", 80_000)
}

func TestSettleMatchRewardsValidatesInput(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisTestStore(t)
	tests := []struct {
		name  string
		input statecontract.SettleMatchRewardsInput
		want  error
	}{
		{
			name:  "missing settlement id",
			input: statecontract.SettleMatchRewardsInput{Rewards: []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}}},
			want:  statecontract.ErrInvalidSettlement,
		},
		{
			name:  "missing rewards",
			input: statecontract.SettleMatchRewardsInput{SettlementID: "room-settlement-3"},
			want:  statecontract.ErrInvalidSettlement,
		},
		{
			name: "invalid player",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 0, Amount: 50}},
			},
			want: statecontract.ErrInvalidPlayer,
		},
		{
			name: "invalid amount",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 0}},
			},
			want: statecontract.ErrInvalidPlayer,
		},
		{
			name: "duplicate player",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards: []statecontract.PlayerCoinReward{
					{PlayerID: 7, Amount: 50},
					{PlayerID: 7, Amount: 75},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
		{
			name: "missing leaderboard map version",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}},
				Leaderboard: &statecontract.MatchLeaderboardRecord{
					Mode:    leaderboardModeSolo,
					Players: []statecontract.PlayerLeaderboardRecord{{PlayerID: 7}},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
		{
			name: "unknown leaderboard mode",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}},
				Leaderboard: &statecontract.MatchLeaderboardRecord{
					Mode:       "squad",
					MapVersion: "wave-v1",
					Players:    []statecontract.PlayerLeaderboardRecord{{PlayerID: 7}},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
		{
			name: "incorrect leaderboard player count",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}},
				Leaderboard: &statecontract.MatchLeaderboardRecord{
					Mode:       leaderboardModeDuo,
					MapVersion: "wave-v1",
					Players:    []statecontract.PlayerLeaderboardRecord{{PlayerID: 7}},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
		{
			name: "leaderboard player differs from reward player",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}},
				Leaderboard: &statecontract.MatchLeaderboardRecord{
					Mode:       leaderboardModeSolo,
					MapVersion: "wave-v1",
					Players:    []statecontract.PlayerLeaderboardRecord{{PlayerID: 8}},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
		{
			name: "negative total kills",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}},
				Leaderboard: &statecontract.MatchLeaderboardRecord{
					Mode:       leaderboardModeSolo,
					MapVersion: "wave-v1",
					Players:    []statecontract.PlayerLeaderboardRecord{{PlayerID: 7, TotalKills: -1}},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
		{
			name: "cleared without positive combat duration",
			input: statecontract.SettleMatchRewardsInput{
				SettlementID: "room-settlement-3",
				Rewards:      []statecontract.PlayerCoinReward{{PlayerID: 7, Amount: 50}},
				Leaderboard: &statecontract.MatchLeaderboardRecord{
					Mode:       leaderboardModeSolo,
					MapVersion: "wave-v1",
					Cleared:    true,
					Players:    []statecontract.PlayerLeaderboardRecord{{PlayerID: 7}},
				},
			},
			want: statecontract.ErrInvalidSettlement,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.SettleMatchRewards(ctx, tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("SettleMatchRewards error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestUpdatePlayerAvatarSuccess(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	want := &statecontract.Player{
		ID:       7,
		Nickname: "Alice",
		Avatar:   "adventurer",
		Email:    "alice@example.com",
		Phone:    "13800000000",
		Coins:    120,
	}
	if err := store.CreatePlayer(ctx, want); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	got, err := store.UpdatePlayerAvatar(ctx, want.ID, "mage")
	if err != nil {
		t.Fatalf("UpdatePlayerAvatar returned error: %v", err)
	}
	if got.ID != want.ID || got.Nickname != want.Nickname || got.Avatar != "mage" ||
		got.Email != want.Email || got.Phone != want.Phone || got.Coins != want.Coins {
		t.Fatalf("updated player = %#v, want original profile with avatar mage", got)
	}
	storedAvatar, err := client.HGet(ctx, playerKey(want.ID), "avatar").Result()
	if err != nil {
		t.Fatalf("HGet avatar returned error: %v", err)
	}
	if storedAvatar != "mage" {
		t.Fatalf("stored avatar = %q, want mage", storedAvatar)
	}
}

func TestUpdatePlayerAvatarValidatesInputAndMissingPlayer(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)

	tests := []struct {
		name     string
		playerID int64
		avatar   string
		want     error
	}{
		{name: "invalid player", playerID: 0, avatar: "mage", want: statecontract.ErrInvalidPlayer},
		{name: "empty avatar", playerID: 7, avatar: "", want: statecontract.ErrInvalidPlayer},
		{name: "missing player", playerID: 7, avatar: "mage", want: statecontract.ErrPlayerNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.UpdatePlayerAvatar(ctx, tt.playerID, tt.avatar)
			if !errors.Is(err, tt.want) {
				t.Fatalf("UpdatePlayerAvatar error = %v, want %v", err, tt.want)
			}
		})
	}

	exists, err := client.Exists(ctx, playerKey(7)).Result()
	if err != nil {
		t.Fatalf("Exists player returned error: %v", err)
	}
	if exists != 0 {
		t.Fatalf("missing player key was created")
	}
}

func TestRegisterAccountCreatesInitialGrowth(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisTestStore(t)

	result, err := store.RegisterAccount(ctx, statecontract.RegisterAccountInput{
		Username:         "alice",
		PasswordHash:     "hash",
		Nickname:         "Alice",
		SessionToken:     "token-1",
		SessionExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("RegisterAccount returned error: %v", err)
	}

	growth, err := store.GetGrowth(ctx, result.Player.ID)
	if err != nil {
		t.Fatalf("GetGrowth returned error: %v", err)
	}
	if growth.PlayerID != result.Player.ID {
		t.Fatalf("growth player id = %d, want %d", growth.PlayerID, result.Player.ID)
	}
	if growth.AttackLevel != 1 || growth.AttackSpeedLevel != 1 || growth.HealthLevel != 1 || growth.MoveSpeedLevel != 1 {
		t.Fatalf("initial growth = %+v, want all levels 1", growth)
	}
}

func TestGrowthUpgradeSuccess(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerAndGrowth(t, ctx, client, 7, 150, &statecontract.Growth{
		PlayerID:         7,
		AttackLevel:      1,
		AttackSpeedLevel: 1,
		HealthLevel:      2,
		MoveSpeedLevel:   1,
	})

	result, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{
		PlayerID:     7,
		UpgradeField: "Attack",
		Cost:         100,
		MaxLevel:     10,
	})
	if err != nil {
		t.Fatalf("UpgradeGrowth returned error: %v", err)
	}
	if result.RemainingCoins != 50 {
		t.Fatalf("remaining coins = %d, want 50", result.RemainingCoins)
	}
	if result.Growth.AttackLevel != 2 {
		t.Fatalf("attack level = %d, want 2", result.Growth.AttackLevel)
	}
	if result.Growth.HealthLevel != 2 {
		t.Fatalf("health level = %d, want unchanged 2", result.Growth.HealthLevel)
	}

	coins, err := client.HGet(ctx, playerKey(7), "coins").Int64()
	if err != nil {
		t.Fatalf("HGet coins returned error: %v", err)
	}
	if coins != 50 {
		t.Fatalf("stored coins = %d, want 50", coins)
	}
	attackLevel, err := client.HGet(ctx, growthKey(7), "attack_level").Int()
	if err != nil {
		t.Fatalf("HGet attack_level returned error: %v", err)
	}
	if attackLevel != 2 {
		t.Fatalf("stored attack level = %d, want 2", attackLevel)
	}
}

func TestGrowthUpgradeRejectsInsufficientCoins(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerAndGrowth(t, ctx, client, 7, 99, &statecontract.Growth{
		PlayerID:         7,
		AttackLevel:      1,
		AttackSpeedLevel: 1,
		HealthLevel:      1,
		MoveSpeedLevel:   1,
	})

	_, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{
		PlayerID:     7,
		UpgradeField: "Attack",
		Cost:         100,
		MaxLevel:     10,
	})
	if !errors.Is(err, statecontract.ErrInsufficientCoins) {
		t.Fatalf("UpgradeGrowth error = %v, want %v", err, statecontract.ErrInsufficientCoins)
	}
	assertGrowthStoredLevel(t, ctx, client, 7, "attack_level", 1)
	assertPlayerCoins(t, ctx, client, 7, 99)
}

func TestGrowthUpgradeRejectsMaxLevel(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)
	seedPlayerAndGrowth(t, ctx, client, 7, 500, &statecontract.Growth{
		PlayerID:         7,
		AttackLevel:      10,
		AttackSpeedLevel: 1,
		HealthLevel:      1,
		MoveSpeedLevel:   1,
	})

	_, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{
		PlayerID:     7,
		UpgradeField: "Attack",
		Cost:         100,
		MaxLevel:     10,
	})
	if !errors.Is(err, statecontract.ErrMaxGrowthLevel) {
		t.Fatalf("UpgradeGrowth error = %v, want %v", err, statecontract.ErrMaxGrowthLevel)
	}
	assertGrowthStoredLevel(t, ctx, client, 7, "attack_level", 10)
	assertPlayerCoins(t, ctx, client, 7, 500)
}

func TestGrowthUpgradeValidatesInputAndMissingState(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)

	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "invalid player",
			run: func() error {
				_, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{PlayerID: 0, UpgradeField: "Attack", Cost: 1, MaxLevel: 10})
				return err
			},
			want: statecontract.ErrInvalidGrowth,
		},
		{
			name: "invalid field",
			run: func() error {
				_, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{PlayerID: 7, UpgradeField: "Critical", Cost: 1, MaxLevel: 10})
				return err
			},
			want: statecontract.ErrInvalidGrowthField,
		},
		{
			name: "missing player",
			run: func() error {
				_, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{PlayerID: 7, UpgradeField: "Attack", Cost: 1, MaxLevel: 10})
				return err
			},
			want: statecontract.ErrPlayerNotFound,
		},
		{
			name: "missing growth",
			run: func() error {
				if err := client.HSet(ctx, playerKey(8), "coins", 100).Err(); err != nil {
					t.Fatalf("seed player coins returned error: %v", err)
				}
				_, err := store.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{PlayerID: 8, UpgradeField: "Attack", Cost: 1, MaxLevel: 10})
				return err
			},
			want: statecontract.ErrGrowthNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGetGrowthValidatesInputAndMissingState(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisTestStore(t)

	if _, err := store.GetGrowth(ctx, 0); !errors.Is(err, statecontract.ErrInvalidGrowth) {
		t.Fatalf("GetGrowth invalid player error = %v, want %v", err, statecontract.ErrInvalidGrowth)
	}
	if _, err := store.GetGrowth(ctx, 7); !errors.Is(err, statecontract.ErrGrowthNotFound) {
		t.Fatalf("GetGrowth missing error = %v, want %v", err, statecontract.ErrGrowthNotFound)
	}
}

type friendRequestWant struct {
	from int64
	to   int64
}

func newRedisTestStore(t *testing.T) (*Store, *redis.Client) {
	t.Helper()

	redisServerPath, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server binary not found")
	}

	addr := freeRedisAddr(t)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split redis address %q: %v", addr, err)
	}

	cmd := exec.Command(
		redisServerPath,
		"--bind", "127.0.0.1",
		"--port", port,
		"--save", "",
		"--appendonly", "no",
		"--dir", t.TempDir(),
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start redis-server: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-done
	})

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("redis-server exited before accepting connections: %v", err)
		case <-ctx.Done():
			t.Fatalf("redis-server did not start: %v", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}

	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("FlushDB returned error: %v", err)
	}
	return NewStore(client), client
}

func freeRedisAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free Redis port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func assertFriendRequests(t *testing.T, got []*statecontract.FriendRequest, want []friendRequestWant) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("friend requests = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].FromPlayerID != want[i].from {
			t.Fatalf("request %d from = %d, want %d", i, got[i].FromPlayerID, want[i].from)
		}
		if got[i].ToPlayerID != want[i].to {
			t.Fatalf("request %d to = %d, want %d", i, got[i].ToPlayerID, want[i].to)
		}
		if got[i].CreatedAt.IsZero() {
			t.Fatalf("request %d created at is zero", i)
		}
	}
}

func assertHashMissing(t *testing.T, ctx context.Context, client *redis.Client, key string) {
	t.Helper()

	values, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		t.Fatalf("HGetAll %s returned error: %v", key, err)
	}
	if len(values) != 0 {
		t.Fatalf("hash %s = %v, want missing", key, values)
	}
}

func assertSetMembers(t *testing.T, ctx context.Context, client *redis.Client, key string, want []string) {
	t.Helper()

	got, err := client.SMembers(ctx, key).Result()
	if err != nil {
		t.Fatalf("SMembers %s returned error: %v", key, err)
	}
	assertStringSet(t, got, want)
}

func assertZSetMembers(t *testing.T, ctx context.Context, client *redis.Client, key string, want []string) {
	t.Helper()

	got, err := client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		t.Fatalf("ZRange %s returned error: %v", key, err)
	}
	assertStringSet(t, got, want)
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("members = %v, want %v", got, want)
	}
	counts := make(map[string]int, len(want))
	for _, value := range want {
		counts[value]++
	}
	for _, value := range got {
		counts[value]--
	}
	for value, count := range counts {
		if count != 0 {
			t.Fatalf("members = %v, want %v; %s count diff %d", got, want, value, count)
		}
	}
}

func assertInt64Set(t *testing.T, got, want []int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	counts := make(map[int64]int, len(want))
	for _, value := range want {
		counts[value]++
	}
	for _, value := range got {
		counts[value]--
	}
	for value, count := range counts {
		if count != 0 {
			t.Fatalf("ids = %v, want %v; %d count diff %d", got, want, value, count)
		}
	}
}

func seedPlayerAndGrowth(t *testing.T, ctx context.Context, client *redis.Client, playerID int64, coins int64, growth *statecontract.Growth) {
	t.Helper()

	seedPlayerWithCoins(t, ctx, client, playerID, coins)
	if err := client.HSet(ctx, growthKey(playerID), map[string]any{
		"player_id":          growth.PlayerID,
		"attack_level":       growth.AttackLevel,
		"attack_speed_level": growth.AttackSpeedLevel,
		"health_level":       growth.HealthLevel,
		"move_speed_level":   growth.MoveSpeedLevel,
	}).Err(); err != nil {
		t.Fatalf("seed growth returned error: %v", err)
	}
}

func seedPlayer(t *testing.T, ctx context.Context, client *redis.Client, playerID int64) {
	t.Helper()

	seedPlayerWithCoins(t, ctx, client, playerID, 0)
}

func seedPlayerWithCoins(t *testing.T, ctx context.Context, client *redis.Client, playerID int64, coins int64) {
	t.Helper()

	if err := client.HSet(ctx, playerKey(playerID), map[string]any{
		"id":       playerID,
		"nickname": "player",
		"coins":    coins,
	}).Err(); err != nil {
		t.Fatalf("seed player returned error: %v", err)
	}
}

func assertPlayerCoins(t *testing.T, ctx context.Context, client *redis.Client, playerID int64, want int64) {
	t.Helper()

	got, err := client.HGet(ctx, playerKey(playerID), "coins").Int64()
	if err != nil {
		t.Fatalf("HGet coins returned error: %v", err)
	}
	if got != want {
		t.Fatalf("coins = %d, want %d", got, want)
	}
}

func assertZScore(t *testing.T, ctx context.Context, client *redis.Client, key, member string, want float64) {
	t.Helper()

	got, err := client.ZScore(ctx, key, member).Result()
	if err != nil {
		t.Fatalf("ZScore %s member %s returned error: %v", key, member, err)
	}
	if got != want {
		t.Fatalf("ZScore %s member %s = %v, want %v", key, member, got, want)
	}
}

func assertRedisKeyMissing(t *testing.T, ctx context.Context, client *redis.Client, key string) {
	t.Helper()

	exists, err := client.Exists(ctx, key).Result()
	if err != nil {
		t.Fatalf("Exists %s returned error: %v", key, err)
	}
	if exists != 0 {
		t.Fatalf("Redis key %s exists, want missing", key)
	}
}

func assertGrowthStoredLevel(t *testing.T, ctx context.Context, client *redis.Client, playerID int64, field string, want int) {
	t.Helper()

	got, err := client.HGet(ctx, growthKey(playerID), field).Int()
	if err != nil {
		t.Fatalf("HGet %s returned error: %v", field, err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", field, got, want)
	}
}
