package grpcserver

import (
	"context"
	"reflect"
	"testing"

	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRegisterBattleNodeAndList(t *testing.T) {
	server := NewServer(newTestCenterService())

	_, err := server.RegisterBattleNode(context.Background(), &rcenterpb.RegisterBattleNodeRequest{
		Node: &rcenterpb.BattleNode{
			Name:          "battle-1",
			KcpAddr:       "127.0.0.1:7001",
			ControlAddr:   "127.0.0.1:9101",
			MaxPlayers:    100,
			ActivePlayers: 3,
		},
	})
	if err != nil {
		t.Fatalf("RegisterBattleNode returned error: %v", err)
	}

	res, err := server.ListBattleNodes(context.Background(), &rcenterpb.ListBattleNodesRequest{})
	if err != nil {
		t.Fatalf("ListBattleNodes returned error: %v", err)
	}
	if len(res.GetNodes()) != 1 {
		t.Fatalf("nodes length = %d, want 1", len(res.GetNodes()))
	}
	node := res.GetNodes()[0]
	if node.GetName() != "battle-1" {
		t.Fatalf("node name = %q, want battle-1", node.GetName())
	}
	if node.GetKcpAddr() != "127.0.0.1:7001" {
		t.Fatalf("node kcp addr = %q, want 127.0.0.1:7001", node.GetKcpAddr())
	}
	if node.GetControlAddr() != "127.0.0.1:9101" {
		t.Fatalf("node control addr = %q, want 127.0.0.1:9101", node.GetControlAddr())
	}
	if node.GetMaxPlayers() != 100 {
		t.Fatalf("node max players = %d, want 100", node.GetMaxPlayers())
	}
	if node.GetActivePlayers() != 3 {
		t.Fatalf("node active players = %d, want 3", node.GetActivePlayers())
	}
	if node.GetLastSeenUnix() == 0 {
		t.Fatalf("node last seen unix is zero")
	}
}

func TestStartMatchCreatesMatchedResult(t *testing.T) {
	server := NewServer(newTestCenterService())
	mustRegisterBattleNode(t, server, &rcenterpb.BattleNode{
		Name:          "battle-1",
		KcpAddr:       "127.0.0.1:7001",
		ControlAddr:   "127.0.0.1:9101",
		MaxPlayers:    100,
		ActivePlayers: 10,
	})
	mustRegisterBattleNode(t, server, &rcenterpb.BattleNode{
		Name:          "battle-2",
		KcpAddr:       "127.0.0.1:7002",
		ControlAddr:   "127.0.0.1:9102",
		MaxPlayers:    100,
		ActivePlayers: 1,
	})

	first, err := server.StartMatch(context.Background(), &rcenterpb.StartMatchRequest{PlayerId: 7})
	if err != nil {
		t.Fatalf("first StartMatch returned error: %v", err)
	}
	if first.GetResult().GetStatus() != string(rcenter.MatchStatusWaiting) {
		t.Fatalf("first status = %q, want %q", first.GetResult().GetStatus(), rcenter.MatchStatusWaiting)
	}

	second, err := server.StartMatch(context.Background(), &rcenterpb.StartMatchRequest{PlayerId: 8})
	if err != nil {
		t.Fatalf("second StartMatch returned error: %v", err)
	}
	result := second.GetResult()
	if result.GetStatus() != string(rcenter.MatchStatusMatched) {
		t.Fatalf("second status = %q, want %q", result.GetStatus(), rcenter.MatchStatusMatched)
	}
	if result.GetRoomName() == "" {
		t.Fatalf("room name is empty")
	}
	if result.GetToken() == "" {
		t.Fatalf("token is empty")
	}
	if result.GetBattleNodeName() != "battle-2" {
		t.Fatalf("battle node name = %q, want battle-2", result.GetBattleNodeName())
	}
	if result.GetBattleKcpAddr() != "127.0.0.1:7002" {
		t.Fatalf("battle kcp addr = %q, want 127.0.0.1:7002", result.GetBattleKcpAddr())
	}
	if !reflect.DeepEqual(result.GetPlayerIds(), []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", result.GetPlayerIds())
	}
}

func TestRegisterBattleNodeInvalidInputMapsToInvalidArgument(t *testing.T) {
	server := NewServer(newTestCenterService())

	_, err := server.RegisterBattleNode(context.Background(), &rcenterpb.RegisterBattleNodeRequest{
		Node: &rcenterpb.BattleNode{
			KcpAddr:    "127.0.0.1:7001",
			MaxPlayers: 100,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("RegisterBattleNode code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestStartMatchInvalidPlayerMapsToInvalidArgument(t *testing.T) {
	server := NewServer(newTestCenterService())

	_, err := server.StartMatch(context.Background(), &rcenterpb.StartMatchRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("StartMatch code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestStartMatchWithoutBattleNodeMapsToUnavailable(t *testing.T) {
	server := NewServer(newTestCenterService())

	_, err := server.StartMatch(context.Background(), &rcenterpb.StartMatchRequest{PlayerId: 7})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("StartMatch code = %v, want %v", status.Code(err), codes.Unavailable)
	}
}

func TestCancelMatch(t *testing.T) {
	server := NewServer(newTestCenterService())
	mustRegisterBattleNode(t, server, &rcenterpb.BattleNode{
		Name:        "battle-1",
		KcpAddr:     "127.0.0.1:7001",
		ControlAddr: "127.0.0.1:9101",
		MaxPlayers:  100,
	})
	if _, err := server.StartMatch(context.Background(), &rcenterpb.StartMatchRequest{PlayerId: 7}); err != nil {
		t.Fatalf("StartMatch returned error: %v", err)
	}

	if _, err := server.CancelMatch(context.Background(), &rcenterpb.CancelMatchRequest{PlayerId: 7}); err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}
}

func TestCancelMatchNotWaitingMapsToFailedPrecondition(t *testing.T) {
	server := NewServer(newTestCenterService())

	_, err := server.CancelMatch(context.Background(), &rcenterpb.CancelMatchRequest{PlayerId: 7})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("CancelMatch code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func mustRegisterBattleNode(t *testing.T, server *Server, node *rcenterpb.BattleNode) {
	t.Helper()
	if _, err := server.RegisterBattleNode(context.Background(), &rcenterpb.RegisterBattleNodeRequest{Node: node}); err != nil {
		t.Fatalf("RegisterBattleNode returned error: %v", err)
	}
}

func newTestCenterService() *rcenter.GameCenterService {
	return rcenter.NewService(&fakeBattleNodeController{})
}

type fakeBattleNodeController struct{}

func (f *fakeBattleNodeController) RegisterNode(ctx context.Context, node rcenter.BattleNode) error {
	return nil
}

func (f *fakeBattleNodeController) CreateRoom(ctx context.Context, nodeName string, input rcenter.CreateBattleRoomInput) error {
	return nil
}
