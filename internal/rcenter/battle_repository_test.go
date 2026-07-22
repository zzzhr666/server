package rcenter

import (
	"context"
	"errors"
	"reflect"
	"testing"

	battlegrpcclient "server/internal/battle/grpcclient"
)

func TestBattleRepositoryRegisterNodeUsesFactory(t *testing.T) {
	factory := &fakeBattleClientFactory{}
	repo := newBattleRepositoryWithFactory(factory.newClient)

	err := repo.RegisterNode(context.Background(), BattleNode{
		Name:        "battle-1",
		ControlAddr: "127.0.0.1:9101",
	})
	if err != nil {
		t.Fatalf("RegisterNode returned error: %v", err)
	}
	if factory.node.Name != "battle-1" {
		t.Fatalf("factory node name = %q, want battle-1", factory.node.Name)
	}
	if factory.node.ControlAddr != "127.0.0.1:9101" {
		t.Fatalf("factory control addr = %q, want 127.0.0.1:9101", factory.node.ControlAddr)
	}
}

func TestBattleRepositoryCreateRoom(t *testing.T) {
	factory := &fakeBattleClientFactory{}
	repo := newBattleRepositoryWithFactory(factory.newClient)
	if err := repo.RegisterNode(context.Background(), BattleNode{Name: "battle-1"}); err != nil {
		t.Fatalf("RegisterNode returned error: %v", err)
	}

	err := repo.CreateRoom(context.Background(), "battle-1", CreateBattleRoomInput{
		RoomName:  "room-1",
		Token:     "token-1",
		PlayerIDs: []int64{7, 8},
	})
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if factory.client.createRoomInput.RoomName != "room-1" {
		t.Fatalf("room name = %q, want room-1", factory.client.createRoomInput.RoomName)
	}
	if factory.client.createRoomInput.Token != "token-1" {
		t.Fatalf("token = %q, want token-1", factory.client.createRoomInput.Token)
	}
	if !reflect.DeepEqual(factory.client.createRoomInput.PlayerIDs, []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", factory.client.createRoomInput.PlayerIDs)
	}
}

func TestBattleRepositoryCreateRoomRequiresRegisteredNode(t *testing.T) {
	repo := newBattleRepositoryWithFactory((&fakeBattleClientFactory{}).newClient)

	err := repo.CreateRoom(context.Background(), "missing", CreateBattleRoomInput{})
	if !errors.Is(err, ErrBattleNodeNotRegistered) {
		t.Fatalf("CreateRoom error = %v, want %v", err, ErrBattleNodeNotRegistered)
	}
}

func TestBattleRepositoryCreateRoomMapsNonOKStatus(t *testing.T) {
	factory := &fakeBattleClientFactory{
		client: &fakeBattleControlClient{
			result: &battlegrpcclient.CreateRoomResult{Status: battlegrpcclient.CreateRoomStatusAlreadyExists},
		},
	}
	repo := newBattleRepositoryWithFactory(factory.newClient)
	if err := repo.RegisterNode(context.Background(), BattleNode{Name: "battle-1"}); err != nil {
		t.Fatalf("RegisterNode returned error: %v", err)
	}

	err := repo.CreateRoom(context.Background(), "battle-1", CreateBattleRoomInput{})
	if !errors.Is(err, ErrCreateBattleRoomFailed) {
		t.Fatalf("CreateRoom error = %v, want %v", err, ErrCreateBattleRoomFailed)
	}
}

func TestBattleRepositoryCreateRoomReturnsClientError(t *testing.T) {
	wantErr := errors.New("battle unavailable")
	factory := &fakeBattleClientFactory{
		client: &fakeBattleControlClient{err: wantErr},
	}
	repo := newBattleRepositoryWithFactory(factory.newClient)
	if err := repo.RegisterNode(context.Background(), BattleNode{Name: "battle-1"}); err != nil {
		t.Fatalf("RegisterNode returned error: %v", err)
	}

	err := repo.CreateRoom(context.Background(), "battle-1", CreateBattleRoomInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateRoom error = %v, want %v", err, wantErr)
	}
}

func TestBattleRepositoryRegisterNodeReplacesExistingClient(t *testing.T) {
	factory := &fakeBattleClientFactory{}
	repo := newBattleRepositoryWithFactory(factory.newClient)
	if err := repo.RegisterNode(context.Background(), BattleNode{Name: "battle-1"}); err != nil {
		t.Fatalf("first RegisterNode returned error: %v", err)
	}
	firstConn := factory.conn

	if err := repo.RegisterNode(context.Background(), BattleNode{Name: "battle-1"}); err != nil {
		t.Fatalf("second RegisterNode returned error: %v", err)
	}
	if !firstConn.closed {
		t.Fatalf("old connection was not closed")
	}
}

func TestBattleRepositoryRegisterNodeReturnsFactoryError(t *testing.T) {
	wantErr := errors.New("dial failed")
	factory := &fakeBattleClientFactory{err: wantErr}
	repo := newBattleRepositoryWithFactory(factory.newClient)

	err := repo.RegisterNode(context.Background(), BattleNode{Name: "battle-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("RegisterNode error = %v, want %v", err, wantErr)
	}
}

type fakeBattleClientFactory struct {
	node   BattleNode
	conn   *fakeBattleConn
	client *fakeBattleControlClient
	err    error
}

func (f *fakeBattleClientFactory) newClient(node BattleNode) (battleConn, battleControlClient, error) {
	f.node = node
	if f.err != nil {
		return nil, nil, f.err
	}
	f.conn = &fakeBattleConn{}
	if f.client == nil {
		f.client = &fakeBattleControlClient{
			result: &battlegrpcclient.CreateRoomResult{Status: battlegrpcclient.CreateRoomStatusOK},
		}
	}
	return f.conn, f.client, nil
}

type fakeBattleConn struct {
	closed bool
}

func (f *fakeBattleConn) Close() error {
	f.closed = true
	return nil
}

type fakeBattleControlClient struct {
	createRoomInput battlegrpcclient.CreateRoomInput
	result          *battlegrpcclient.CreateRoomResult
	err             error
}

func (f *fakeBattleControlClient) CreateRoom(ctx context.Context, input battlegrpcclient.CreateRoomInput) (*battlegrpcclient.CreateRoomResult, error) {
	f.createRoomInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}
