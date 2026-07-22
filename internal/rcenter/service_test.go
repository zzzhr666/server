package rcenter

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestServiceRegisterBattleNode(t *testing.T) {
	battleRooms := &fakeBattleRoomCreator{}
	svc := NewService(battleRooms)

	err := svc.RegisterBattleNode(context.Background(), BattleNode{
		Name:        "battle-1",
		KCPAddr:     "127.0.0.1:7001",
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
		KCPAddr:    "127.0.0.1:7001",
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
		KCPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	result, err := svc.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("StartMatch returned error: %v", err)
	}
	if result.Status != MatchStatusWaiting {
		t.Fatalf("status = %q, want %q", result.Status, MatchStatusWaiting)
	}
	if result.RoomName != "" || result.Token != "" || result.BattleKCPAddr != "" {
		t.Fatalf("waiting result should not include room data: %+v", result)
	}
}

func TestServiceStartMatchCreatesRoomForSecondPlayer(t *testing.T) {
	battleRooms := &fakeBattleRoomCreator{}
	svc := NewService(battleRooms)
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:          "battle-1",
		KCPAddr:       "127.0.0.1:7001",
		ControlAddr:   "127.0.0.1:9101",
		MaxPlayers:    100,
		ActivePlayers: 10,
	})
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:          "battle-2",
		KCPAddr:       "127.0.0.1:7002",
		ControlAddr:   "127.0.0.1:9102",
		MaxPlayers:    100,
		ActivePlayers: 1,
	})

	first, err := svc.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	second, err := svc.StartMatch(context.Background(), 8)
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
	if second.BattleKCPAddr != "127.0.0.1:7002" {
		t.Fatalf("battle kcp addr = %q, want 127.0.0.1:7002", second.BattleKCPAddr)
	}
	if !reflect.DeepEqual(second.PlayerIDs, []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", second.PlayerIDs)
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
}

func TestServiceStartMatchReturnsCreateRoomError(t *testing.T) {
	wantErr := errors.New("battle create failed")
	battleRooms := &fakeBattleRoomCreator{createRoomErr: wantErr}
	svc := NewService(battleRooms)
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		KCPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	first, err := svc.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	_, err = svc.StartMatch(context.Background(), 8)
	if !errors.Is(err, wantErr) {
		t.Fatalf("second StartMatch error = %v, want %v", err, wantErr)
	}
}

func TestServiceStartMatchDoesNotQueueSamePlayerTwice(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		KCPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	first, err := svc.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	second, err := svc.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}
	if second.Status != MatchStatusWaiting {
		t.Fatalf("second status = %q, want %q", second.Status, MatchStatusWaiting)
	}

	third, err := svc.StartMatch(context.Background(), 8)
	if err != nil {
		t.Fatalf("third StartMatch returned error: %v", err)
	}
	if third.Status != MatchStatusMatched {
		t.Fatalf("third status = %q, want %q", third.Status, MatchStatusMatched)
	}
}

func TestServiceStartMatchInvalidPlayer(t *testing.T) {
	svc := newTestService()

	_, err := svc.StartMatch(context.Background(), 0)
	if !errors.Is(err, ErrInvalidPlayerID) {
		t.Fatalf("StartMatch error = %v, want %v", err, ErrInvalidPlayerID)
	}
}

func TestServiceStartMatchWithoutBattleNode(t *testing.T) {
	svc := newTestService()

	_, err := svc.StartMatch(context.Background(), 7)
	if !errors.Is(err, ErrNoAvailableBattleNode) {
		t.Fatalf("StartMatch error = %v, want %v", err, ErrNoAvailableBattleNode)
	}
}

func TestServiceCancelMatchRemovesWaitingPlayer(t *testing.T) {
	svc := newTestService()
	mustRegisterBattleNode(t, svc, BattleNode{
		Name:        "battle-1",
		KCPAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})

	first, err := svc.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.Status != MatchStatusWaiting {
		t.Fatalf("first status = %q, want %q", first.Status, MatchStatusWaiting)
	}

	if err := svc.CancelMatch(context.Background(), 7); err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}

	second, err := svc.StartMatch(context.Background(), 8)
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
	return NewService(&fakeBattleRoomCreator{})
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
