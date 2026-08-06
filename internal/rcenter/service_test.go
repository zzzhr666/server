package rcenter

import (
	"context"
	"errors"
	"reflect"
	statecontract "server/internal/contract/state"
	"testing"
)

func TestServiceRegisterBattleNode(t *testing.T) {
	battleRooms := &fakeBattleRoomCreator{}
	svc := NewService(ServiceConfig{
		BattleNodeController: battleRooms,
		RewardRule:           DefaultRewardRule(),
		GrowthClient: &fakeGrowthClient{growths: map[int64]*statecontract.Growth{
			7: {
				PlayerID:         7,
				AttackLevel:      2,
				AttackSpeedLevel: 3,
				HealthLevel:      4,
				MoveSpeedLevel:   5,
			},
			8: {
				PlayerID:         8,
				AttackLevel:      6,
				AttackSpeedLevel: 7,
				HealthLevel:      8,
				MoveSpeedLevel:   9,
			},
		}},
	})

	err := svc.RegisterBattleNode(context.Background(), BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})
	if err != nil {
		t.Fatalf("RegisterBattleNode returned error: %v", err)
	}

	nodes := svc.ListBattleNodes()
	if len(nodes) != 1 {
		t.Fatalf("nodes length = %d, want 1", len(nodes))
	}
	if nodes[0].Name != "battle-1" {
		t.Fatalf("node name = %q, want battle-1", nodes[0].Name)
	}
	if nodes[0].LastSeen.IsZero() {
		t.Fatalf("last seen is zero")
	}
	if battleRooms.registerNodeInput.Name != "battle-1" {
		t.Fatalf("registered battle node name = %q, want battle-1", battleRooms.registerNodeInput.Name)
	}
	if battleRooms.registerNodeInput.ControlAddr != "127.0.0.1:9101" {
		t.Fatalf("registered battle control addr = %q, want 127.0.0.1:9101", battleRooms.registerNodeInput.ControlAddr)
	}
}

func TestServiceRegisterBattleNodeInvalidInput(t *testing.T) {
	svc := newTestService()

	err := svc.RegisterBattleNode(context.Background(), BattleNode{
		Name:       "",
		UDPAddr:    "127.0.0.1:7001",
		MaxPlayers: 100,
	})
	if !errors.Is(err, ErrInvalidBattleNode) {
		t.Fatalf("RegisterBattleNode error = %v, want %v", err, ErrInvalidBattleNode)
	}
	if len(svc.ListBattleNodes()) != 0 {
		t.Fatalf("registered invalid battle node")
	}
}

func TestServiceStartMatchWaitsForFirstPlayer(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	result, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("StartMatch returned error: %v", err)
	}
	if result.Status != MatchStatusWaiting {
		t.Fatalf("status = %q, want %q", result.Status, MatchStatusWaiting)
	}
	if result.RoomName != "" || result.Token != "" || result.BattleUDPAddr != "" {
		t.Fatalf("waiting result should not include room data: %+v", result)
	}
}

func TestServiceStartMatchCreatesRoomForSecondPlayer(t *testing.T) {
	battleRooms := &fakeBattleRoomCreator{}
	svc := NewService(ServiceConfig{
		BattleNodeController: battleRooms,
		RewardRule:           DefaultRewardRule(),
		GrowthClient: &fakeGrowthClient{growths: map[int64]*statecontract.Growth{
			7: {
				PlayerID:         7,
				AttackLevel:      2,
				AttackSpeedLevel: 3,
				HealthLevel:      4,
				MoveSpeedLevel:   5,
			},
			8: {
				PlayerID:         8,
				AttackLevel:      6,
				AttackSpeedLevel: 7,
				HealthLevel:      8,
				MoveSpeedLevel:   9,
			},
		}},
	})
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:          "battle-1",
		UDPAddr:       "127.0.0.1:7001",
		ControlAddr:   "127.0.0.1:9101",
		MaxPlayers:    100,
		ActivePlayers: 10,
	})
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:          "battle-2",
		UDPAddr:       "127.0.0.1:7002",
		ControlAddr:   "127.0.0.1:9102",
		MaxPlayers:    100,
		ActivePlayers: 1,
	})

	first, err := svc.StartMatch(context.Background(), 7, "axe")
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	second, err := svc.StartMatch(context.Background(), 8, "dagger")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}
	if second.Status != MatchStatusMatched {
		t.Fatalf("second status = %q, want %q", second.Status, MatchStatusMatched)
	}
	if second.RoomName == "" {
		t.Fatalf("room id is empty")
	}
	if second.Token == "" {
		t.Fatalf("token is empty")
	}
	if second.BattleNodeName != "battle-2" {
		t.Fatalf("battle node name = %q, want battle-2", second.BattleNodeName)
	}
	if second.BattleUDPAddr != "127.0.0.1:7002" {
		t.Fatalf("battle udp addr = %q, want 127.0.0.1:7002", second.BattleUDPAddr)
	}
	if !reflect.DeepEqual(second.PlayerIDs, []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", second.PlayerIDs)
	}
	wantLoadouts := []PlayerLoadout{
		{PlayerID: 7, Weapon: "axe", AttackLevel: 2, AttackSpeedLevel: 3, HealthLevel: 4, MoveSpeedLevel: 5},
		{PlayerID: 8, Weapon: "dagger", AttackLevel: 6, AttackSpeedLevel: 7, HealthLevel: 8, MoveSpeedLevel: 9},
	}
	if !reflect.DeepEqual(second.PlayerLoadouts, wantLoadouts) {
		t.Fatalf("player loadouts = %+v, want %+v", second.PlayerLoadouts, wantLoadouts)
	}
	if battleRooms.createRoomNodeName != "battle-2" {
		t.Fatalf("battle create room node = %q, want battle-2", battleRooms.createRoomNodeName)
	}
	if battleRooms.createRoomInput.RoomName != second.RoomName {
		t.Fatalf("battle room name = %q, want %q", battleRooms.createRoomInput.RoomName, second.RoomName)
	}
	if battleRooms.createRoomInput.Token != second.Token {
		t.Fatalf("battle token = %q, want %q", battleRooms.createRoomInput.Token, second.Token)
	}
	if !reflect.DeepEqual(battleRooms.createRoomInput.PlayerIDs, []int64{7, 8}) {
		t.Fatalf("battle player ids = %v, want [7 8]", battleRooms.createRoomInput.PlayerIDs)
	}
	if !reflect.DeepEqual(battleRooms.createRoomInput.PlayerLoadouts, wantLoadouts) {
		t.Fatalf("battle player loadouts = %+v, want %+v", battleRooms.createRoomInput.PlayerLoadouts, wantLoadouts)
	}
}

func TestServiceStartMatchStoresActiveMatchForAllPlayers(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	if _, err := svc.StartMatch(context.Background(), 7, "axe"); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	matched, err := svc.StartMatch(context.Background(), 8, "dagger")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	for _, playerID := range matched.PlayerIDs {
		activeMatch, ok := svc.activeMatches[playerID]
		if !ok {
			t.Fatalf("active match missing for player %d", playerID)
		}
		if activeMatch.RoomName != matched.RoomName || activeMatch.Token != matched.Token ||
			activeMatch.BattleNodeName != matched.BattleNodeName || activeMatch.BattleUDPAddr != matched.BattleUDPAddr {
			t.Fatalf("active match = %+v, want match result data", activeMatch)
		}
		if !reflect.DeepEqual(activeMatch.PlayerIDs, matched.PlayerIDs) {
			t.Fatalf("active player ids = %v, want %v", activeMatch.PlayerIDs, matched.PlayerIDs)
		}
		if !reflect.DeepEqual(activeMatch.PlayerLoadouts, matched.PlayerLoadouts) {
			t.Fatalf("active loadouts = %+v, want %+v", activeMatch.PlayerLoadouts, matched.PlayerLoadouts)
		}
		if activeMatch.CreatedAt.IsZero() {
			t.Fatal("active match creation time is zero")
		}
	}

	matched.PlayerIDs[0] = 99
	matched.PlayerLoadouts[0].Weapon = "changed"
	activeMatch := svc.activeMatches[7]
	if activeMatch.PlayerIDs[0] != 7 || activeMatch.PlayerLoadouts[0].Weapon != "axe" {
		t.Fatalf("active match shares slices with returned result: %+v", activeMatch)
	}
}

func TestServiceResumeMatchReturnsActiveMatch(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})
	if _, err := svc.StartMatch(context.Background(), 7, "axe"); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	matched, err := svc.StartMatch(context.Background(), 8, "dagger")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	resumed, err := svc.ResumeMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("ResumeMatch returned error: %v", err)
	}
	if !reflect.DeepEqual(resumed, matched) {
		t.Fatalf("resumed match = %+v, want %+v", resumed, matched)
	}

	resumed.PlayerIDs[0] = 99
	resumed.PlayerLoadouts[0].Weapon = "changed"
	activeMatch := svc.activeMatches[7]
	if activeMatch.PlayerIDs[0] != 7 || activeMatch.PlayerLoadouts[0].Weapon != "axe" {
		t.Fatalf("active match shares slices with resumed result: %+v", activeMatch)
	}
}

func TestServiceResumeMatchRejectsMissingActiveMatch(t *testing.T) {
	_, err := newTestService().ResumeMatch(context.Background(), 7)
	if !errors.Is(err, ErrActiveMatchNotFound) {
		t.Fatalf("ResumeMatch error = %v, want %v", err, ErrActiveMatchNotFound)
	}
}

func TestServiceResumeMatchRejectsInvalidPlayer(t *testing.T) {
	_, err := newTestService().ResumeMatch(context.Background(), 0)
	if !errors.Is(err, ErrInvalidPlayerID) {
		t.Fatalf("ResumeMatch error = %v, want %v", err, ErrInvalidPlayerID)
	}
}

func TestServiceStartMatchReturnsCreateRoomError(t *testing.T) {
	wantErr := errors.New("battle create failed")
	battleRooms := &fakeBattleRoomCreator{createRoomErr: wantErr}
	svc := NewService(ServiceConfig{
		BattleNodeController: battleRooms,
		RewardRule:           DefaultRewardRule(),
		GrowthClient:         &fakeGrowthClient{},
	})
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	first, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	_, err = svc.StartMatch(context.Background(), 8, "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("second StartMatch error = %v, want %v", err, wantErr)
	}

	battleRooms.createRoomErr = nil
	third, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("third StartMatch returned error: %v", err)
	}
	if third.Status != MatchStatusWaiting {
		t.Fatalf("third status = %q, want %q", third.Status, MatchStatusWaiting)
	}
}

func TestServiceStartMatchDoesNotQueueSamePlayerTwice(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	first, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	second, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}
	if second.Status != MatchStatusWaiting {
		t.Fatalf("second status = %q, want %q", second.Status, MatchStatusWaiting)
	}

	third, err := svc.StartMatch(context.Background(), 8, "")
	if err != nil {
		t.Fatalf("third StartMatch returned error: %v", err)
	}
	if third.Status != MatchStatusMatched {
		t.Fatalf("third status = %q, want %q", third.Status, MatchStatusMatched)
	}
}

func TestServiceStartMatchRejectsPlayerInGame(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	if _, err := svc.StartMatch(context.Background(), 7, ""); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if _, err := svc.StartMatch(context.Background(), 8, ""); err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	_, err := svc.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, ErrPlayerInGame) {
		t.Fatalf("third StartMatch error = %v, want %v", err, ErrPlayerInGame)
	}

	result, err := svc.StartMatch(context.Background(), 9, "")
	if err != nil {
		t.Fatalf("fourth StartMatch returned error after in-game rejection: %v", err)
	}
	if result.Status != MatchStatusWaiting {
		t.Fatalf("fourth status = %q, want %q", result.Status, MatchStatusWaiting)
	}
}

func TestServiceFinishMatchAllowsPlayersToMatchAgain(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	if _, err := svc.StartMatch(context.Background(), 7, ""); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	matched, err := svc.StartMatch(context.Background(), 8, "")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	if err := svc.FinishMatch(context.Background(), FinishMatchInput{PlayerIDs: matched.PlayerIDs}); err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	for _, playerID := range matched.PlayerIDs {
		if _, ok := svc.activeMatches[playerID]; ok {
			t.Fatalf("active match remains for player %d after FinishMatch", playerID)
		}
	}

	result, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("StartMatch after FinishMatch returned error: %v", err)
	}
	if result.Status != MatchStatusWaiting {
		t.Fatalf("status = %q, want %q", result.Status, MatchStatusWaiting)
	}
}

func TestServiceFinishMatchAddsCoinRewards(t *testing.T) {
	coins := &fakeCoinClient{}
	svc := NewService(ServiceConfig{
		BattleNodeController: &fakeBattleRoomCreator{},
		CoinClient:           coins,
		GrowthClient:         &fakeGrowthClient{},
		RewardRule: RewardRule{
			BaseReward:    50,
			VictoryReward: 100,
			MonsterKillReward: map[string]int64{
				"melee": 10,
				"elite": 35,
			},
		},
	})
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	if _, err := svc.StartMatch(context.Background(), 7, ""); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	matched, err := svc.StartMatch(context.Background(), 8, "")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	err = svc.FinishMatch(context.Background(), FinishMatchInput{
		PlayerIDs: matched.PlayerIDs,
		Reason:    BattleFinishReasonVictory,
		PlayerStats: []PlayerBattleStats{
			{
				PlayerID:   7,
				TotalKills: 2,
				Kills: []MonsterKillCount{
					{MonsterKind: "melee", Count: 2},
				},
			},
			{
				PlayerID:   8,
				TotalKills: 1,
				Kills: []MonsterKillCount{
					{MonsterKind: "elite", Count: 1},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	wantInputs := []statecontract.AddPlayerCoinsInput{
		{PlayerID: 7, Amount: 170},
		{PlayerID: 8, Amount: 185},
	}
	if !reflect.DeepEqual(coins.inputs, wantInputs) {
		t.Fatalf("coin inputs = %+v, want %+v", coins.inputs, wantInputs)
	}

	result, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("StartMatch after FinishMatch returned error: %v", err)
	}
	if result.Status != MatchStatusWaiting {
		t.Fatalf("status = %q, want %q", result.Status, MatchStatusWaiting)
	}
}

func TestServiceFinishMatchRejectsRewardsWithoutCoinClient(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	if _, err := svc.StartMatch(context.Background(), 7, ""); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	matched, err := svc.StartMatch(context.Background(), 8, "")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	err = svc.FinishMatch(context.Background(), FinishMatchInput{
		PlayerIDs: matched.PlayerIDs,
		PlayerStats: []PlayerBattleStats{
			{PlayerID: 7},
		},
	})
	if !errors.Is(err, ErrUnavailableCoinClient) {
		t.Fatalf("FinishMatch error = %v, want %v", err, ErrUnavailableCoinClient)
	}

	_, err = svc.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, ErrPlayerInGame) {
		t.Fatalf("StartMatch after failed FinishMatch error = %v, want %v", err, ErrPlayerInGame)
	}
}

func TestServiceFinishMatchReturnsCoinErrorWithoutReleasingPlayers(t *testing.T) {
	wantErr := statecontract.ErrPlayerNotFound
	svc := NewService(ServiceConfig{
		BattleNodeController: &fakeBattleRoomCreator{},
		CoinClient:           &fakeCoinClient{err: wantErr},
		RewardRule:           DefaultRewardRule(),
		GrowthClient:         &fakeGrowthClient{},
	})
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	if _, err := svc.StartMatch(context.Background(), 7, ""); err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	matched, err := svc.StartMatch(context.Background(), 8, "")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}

	err = svc.FinishMatch(context.Background(), FinishMatchInput{
		PlayerIDs: matched.PlayerIDs,
		PlayerStats: []PlayerBattleStats{
			{PlayerID: 7},
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("FinishMatch error = %v, want %v", err, wantErr)
	}

	_, err = svc.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, ErrPlayerInGame) {
		t.Fatalf("StartMatch after failed FinishMatch error = %v, want %v", err, ErrPlayerInGame)
	}
}

func TestServiceFinishMatchInvalidPlayer(t *testing.T) {
	svc := newTestService()

	err := svc.FinishMatch(context.Background(), FinishMatchInput{PlayerIDs: []int64{7, 0}})
	if !errors.Is(err, ErrInvalidPlayerID) {
		t.Fatalf("FinishMatch error = %v, want %v", err, ErrInvalidPlayerID)
	}
}

func TestServiceStartMatchInvalidPlayer(t *testing.T) {
	svc := newTestService()

	_, err := svc.StartMatch(context.Background(), 0, "")
	if !errors.Is(err, ErrInvalidPlayerID) {
		t.Fatalf("StartMatch error = %v, want %v", err, ErrInvalidPlayerID)
	}
}

func TestServiceStartMatchWithoutBattleNode(t *testing.T) {
	svc := newTestService()

	_, err := svc.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, ErrNoAvailableBattleNode) {
		t.Fatalf("StartMatch error = %v, want %v", err, ErrNoAvailableBattleNode)
	}
}

func TestServiceCancelMatchRemovesWaitingPlayer(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		UDPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	first, err := svc.StartMatch(context.Background(), 7, "")
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	if err := svc.CancelMatch(context.Background(), 7); err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}

	second, err := svc.StartMatch(context.Background(), 8, "")
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}
	if second.Status != MatchStatusWaiting {
		t.Fatalf("second status = %q, want %q", second.Status, MatchStatusWaiting)
	}
}

func TestServiceCancelMatchNotWaiting(t *testing.T) {
	svc := newTestService()

	err := svc.CancelMatch(context.Background(), 7)
	if !errors.Is(err, ErrPlayerNotWaiting) {
		t.Fatalf("CancelMatch error = %v, want %v", err, ErrPlayerNotWaiting)
	}
}

func TestServiceCancelMatchInvalidPlayer(t *testing.T) {
	svc := newTestService()

	err := svc.CancelMatch(context.Background(), 0)
	if !errors.Is(err, ErrInvalidPlayerID) {
		t.Fatalf("CancelMatch error = %v, want %v", err, ErrInvalidPlayerID)
	}
}

func mustRegisterBattleNode(t *testing.T, svc *GameCenterService, node BattleNode) {
	t.Helper()
	if err := svc.RegisterBattleNode(context.Background(), node); err != nil {
		t.Fatalf("RegisterBattleNode returned error: %v", err)
	}
}

func newTestService() *GameCenterService {
	return NewService(ServiceConfig{
		BattleNodeController: &fakeBattleRoomCreator{},
		RewardRule:           DefaultRewardRule(),
		GrowthClient:         &fakeGrowthClient{},
	})
}

type fakeBattleRoomCreator struct {
	registerNodeInput  BattleNode
	createRoomNodeName string
	createRoomInput    CreateBattleRoomInput
	registerNodeErr    error
	createRoomErr      error
}

func (f *fakeBattleRoomCreator) RegisterNode(ctx context.Context, node BattleNode) error {
	f.registerNodeInput = node
	return f.registerNodeErr
}

func (f *fakeBattleRoomCreator) CreateRoom(ctx context.Context, nodeName string, input CreateBattleRoomInput) error {
	f.createRoomNodeName = nodeName
	f.createRoomInput = input
	return f.createRoomErr
}

type fakeCoinClient struct {
	inputs []statecontract.AddPlayerCoinsInput
	err    error
}

func (f *fakeCoinClient) AddPlayerCoins(_ context.Context, input statecontract.AddPlayerCoinsInput) (*statecontract.AddPlayerCoinsResult, error) {
	f.inputs = append(f.inputs, input)
	if f.err != nil {
		return nil, f.err
	}
	return &statecontract.AddPlayerCoinsResult{
		PlayerID: input.PlayerID,
		Coins:    input.Amount,
	}, nil
}

type fakeGrowthClient struct {
	growths map[int64]*statecontract.Growth
	err     error
}

func (f *fakeGrowthClient) GetGrowth(_ context.Context, playerID int64) (*statecontract.Growth, error) {
	if f.err != nil {
		return nil, f.err
	}
	if growth := f.growths[playerID]; growth != nil {
		return growth, nil
	}
	return &statecontract.Growth{
		PlayerID:         playerID,
		AttackLevel:      1,
		AttackSpeedLevel: 1,
		HealthLevel:      1,
		MoveSpeedLevel:   1,
	}, nil
}

func (f *fakeGrowthClient) UpgradeGrowth(_ context.Context, input statecontract.UpgradeGrowthInput) (*statecontract.UpgradeGrowthResult, error) {
	return nil, errors.New("UpgradeGrowth is not used by these tests")
}
