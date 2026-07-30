package grpcclient

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClientStartMatch(t *testing.T) {
	grpcCenter := &fakeRCenterServiceClient{
		startMatchResponse: &rcenterpb.StartMatchResponse{
			Result: &rcenterpb.MatchResult{
				Status:         string(rcenter.MatchStatusMatched),
				RoomName:       "room-1",
				Token:          "token-1",
				BattleNodeName: "battle-1",
				BattleKcpAddr:  "127.0.0.1:7001",
				PlayerIds:      []int64{7, 8},
				PlayerLoadouts: []*rcenterpb.PlayerLoadout{
					{PlayerId: 7, Weapon: "axe"},
					{PlayerId: 8, Weapon: "dagger"},
				},
			},
		},
	}
	client := NewClient(grpcCenter)

	result, err := client.StartMatch(context.Background(), 7, "axe")
	if err != nil {
		t.Fatalf("StartMatch returned error: %v", err)
	}
	if grpcCenter.startMatchRequest.GetPlayerId() != 7 {
		t.Fatalf("player id = %d, want 7", grpcCenter.startMatchRequest.GetPlayerId())
	}
	if grpcCenter.startMatchRequest.GetWeapon() != "axe" {
		t.Fatalf("weapon = %q, want axe", grpcCenter.startMatchRequest.GetWeapon())
	}
	if result.Status != rcenter.MatchStatusMatched {
		t.Fatalf("status = %q, want %q", result.Status, rcenter.MatchStatusMatched)
	}
	if result.RoomName != "room-1" {
		t.Fatalf("room name = %q, want room-1", result.RoomName)
	}
	if result.Token != "token-1" {
		t.Fatalf("token = %q, want token-1", result.Token)
	}
	if result.BattleNodeName != "battle-1" {
		t.Fatalf("battle node name = %q, want battle-1", result.BattleNodeName)
	}
	if result.BattleKCPAddr != "127.0.0.1:7001" {
		t.Fatalf("battle kcp addr = %q, want 127.0.0.1:7001", result.BattleKCPAddr)
	}
	if !reflect.DeepEqual(result.PlayerIDs, []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", result.PlayerIDs)
	}
	wantLoadouts := []rcenter.PlayerLoadout{
		{PlayerID: 7, Weapon: "axe"},
		{PlayerID: 8, Weapon: "dagger"},
	}
	if !reflect.DeepEqual(result.PlayerLoadouts, wantLoadouts) {
		t.Fatalf("player loadouts = %+v, want %+v", result.PlayerLoadouts, wantLoadouts)
	}
}

func TestClientStartMatchMapsInvalidPlayer(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.InvalidArgument, rcenter.ErrInvalidPlayerID.Error()),
	})

	_, err := client.StartMatch(context.Background(), 0, "")
	if !errors.Is(err, rcenter.ErrInvalidPlayerID) {
		t.Fatalf("StartMatch error = %v, want %v", err, rcenter.ErrInvalidPlayerID)
	}
}

func TestClientStartMatchMapsNoAvailableBattleNode(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.Unavailable, rcenter.ErrNoAvailableBattleNode.Error()),
	})

	_, err := client.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, rcenter.ErrNoAvailableBattleNode) {
		t.Fatalf("StartMatch error = %v, want %v", err, rcenter.ErrNoAvailableBattleNode)
	}
}

func TestClientStartMatchMapsUnavailableGrowthClient(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.Unavailable, rcenter.ErrUnavailableGrowthClient.Error()),
	})

	_, err := client.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, rcenter.ErrUnavailableGrowthClient) {
		t.Fatalf("StartMatch error = %v, want %v", err, rcenter.ErrUnavailableGrowthClient)
	}
}

func TestClientFinishMatchMapsUnavailableCoinClient(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.Unavailable, rcenter.ErrUnavailableCoinClient.Error()),
	})

	err := client.FinishMatch(context.Background(), rcenter.FinishMatchInput{
		PlayerIDs: []int64{7},
		PlayerStats: []rcenter.PlayerBattleStats{
			{PlayerID: 7},
		},
	})
	if !errors.Is(err, rcenter.ErrUnavailableCoinClient) {
		t.Fatalf("FinishMatch error = %v, want %v", err, rcenter.ErrUnavailableCoinClient)
	}
}

func TestClientCancelMatch(t *testing.T) {
	grpcCenter := &fakeRCenterServiceClient{}
	client := NewClient(grpcCenter)

	err := client.CancelMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}
	if grpcCenter.cancelMatchRequest.GetPlayerId() != 7 {
		t.Fatalf("player id = %d, want 7", grpcCenter.cancelMatchRequest.GetPlayerId())
	}
}

func TestClientCancelMatchMapsPlayerNotWaiting(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.FailedPrecondition, rcenter.ErrPlayerNotWaiting.Error()),
	})

	err := client.CancelMatch(context.Background(), 7)
	if !errors.Is(err, rcenter.ErrPlayerNotWaiting) {
		t.Fatalf("CancelMatch error = %v, want %v", err, rcenter.ErrPlayerNotWaiting)
	}
}

func TestClientStartMatchMapsPlayerInGame(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.FailedPrecondition, rcenter.ErrPlayerInGame.Error()),
	})

	_, err := client.StartMatch(context.Background(), 7, "")
	if !errors.Is(err, rcenter.ErrPlayerInGame) {
		t.Fatalf("StartMatch error = %v, want %v", err, rcenter.ErrPlayerInGame)
	}
}

func TestClientFinishMatch(t *testing.T) {
	grpcCenter := &fakeRCenterServiceClient{}
	client := NewClient(grpcCenter)

	err := client.FinishMatch(context.Background(), rcenter.FinishMatchInput{
		PlayerIDs: []int64{7, 8},
		Reason:    rcenter.BattleFinishReasonVictory,
		PlayerStats: []rcenter.PlayerBattleStats{
			{
				PlayerID:   7,
				TotalKills: 3,
				Kills: []rcenter.MonsterKillCount{
					{MonsterKind: "melee", Count: 2},
					{MonsterKind: "elite", Count: 1},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	if !reflect.DeepEqual(grpcCenter.finishMatchRequest.GetPlayerIds(), []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", grpcCenter.finishMatchRequest.GetPlayerIds())
	}
	if grpcCenter.finishMatchRequest.GetReason() != rcenter.BattleFinishReasonVictory {
		t.Fatalf("reason = %q, want %q", grpcCenter.finishMatchRequest.GetReason(), rcenter.BattleFinishReasonVictory)
	}
	stats := grpcCenter.finishMatchRequest.GetPlayerStats()
	if len(stats) != 1 {
		t.Fatalf("player stats = %d, want 1", len(stats))
	}
	if stats[0].GetPlayerId() != 7 {
		t.Fatalf("stats player id = %d, want 7", stats[0].GetPlayerId())
	}
	if stats[0].GetTotalKills() != 3 {
		t.Fatalf("stats total kills = %d, want 3", stats[0].GetTotalKills())
	}
	if len(stats[0].GetKills()) != 2 {
		t.Fatalf("stats kills = %d, want 2", len(stats[0].GetKills()))
	}
	if stats[0].GetKills()[0].GetMonsterKind() != "melee" || stats[0].GetKills()[0].GetCount() != 2 {
		t.Fatalf("first kill = %+v, want melee x2", stats[0].GetKills()[0])
	}
}

func TestClientRegisterBattleNode(t *testing.T) {
	grpcCenter := &fakeRCenterServiceClient{}
	client := NewClient(grpcCenter)

	err := client.RegisterBattleNode(context.Background(), rcenter.BattleNode{
		Name:          "battle-1",
		KCPAddr:       "127.0.0.1:7001",
		ControlAddr:   "127.0.0.1:9101",
		MaxPlayers:    100,
		ActivePlayers: 3,
	})
	if err != nil {
		t.Fatalf("RegisterBattleNode returned error: %v", err)
	}
	node := grpcCenter.registerBattleNodeRequest.GetNode()
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
}

func TestClientRegisterBattleNodeMapsInvalidInput(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		err: status.Error(codes.InvalidArgument, rcenter.ErrInvalidBattleNode.Error()),
	})

	err := client.RegisterBattleNode(context.Background(), rcenter.BattleNode{})
	if !errors.Is(err, rcenter.ErrInvalidBattleNode) {
		t.Fatalf("RegisterBattleNode error = %v, want %v", err, rcenter.ErrInvalidBattleNode)
	}
}

func TestClientListBattleNodes(t *testing.T) {
	client := NewClient(&fakeRCenterServiceClient{
		listBattleNodesResponse: &rcenterpb.ListBattleNodesResponse{
			Nodes: []*rcenterpb.BattleNode{
				{
					Name:          "battle-1",
					KcpAddr:       "127.0.0.1:7001",
					ControlAddr:   "127.0.0.1:9101",
					MaxPlayers:    100,
					ActivePlayers: 3,
					LastSeenUnix:  123,
				},
			},
		},
	})

	nodes, err := client.ListBattleNodes(context.Background())
	if err != nil {
		t.Fatalf("ListBattleNodes returned error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("nodes length = %d, want 1", len(nodes))
	}
	node := nodes[0]
	if node.Name != "battle-1" {
		t.Fatalf("node name = %q, want battle-1", node.Name)
	}
	if node.KCPAddr != "127.0.0.1:7001" {
		t.Fatalf("node kcp addr = %q, want 127.0.0.1:7001", node.KCPAddr)
	}
	if node.ControlAddr != "127.0.0.1:9101" {
		t.Fatalf("node control addr = %q, want 127.0.0.1:9101", node.ControlAddr)
	}
	if node.MaxPlayers != 100 {
		t.Fatalf("node max players = %d, want 100", node.MaxPlayers)
	}
	if node.ActivePlayers != 3 {
		t.Fatalf("node active players = %d, want 3", node.ActivePlayers)
	}
	if got := node.LastSeen.Unix(); got != 123 {
		t.Fatalf("node last seen unix = %d, want 123", got)
	}
}

type fakeRCenterServiceClient struct {
	rcenterpb.UnimplementedRCenterServiceServer

	err                       error
	registerBattleNodeRequest *rcenterpb.RegisterBattleNodeRequest
	listBattleNodesResponse   *rcenterpb.ListBattleNodesResponse
	startMatchRequest         *rcenterpb.StartMatchRequest
	startMatchResponse        *rcenterpb.StartMatchResponse
	cancelMatchRequest        *rcenterpb.CancelMatchRequest
	finishMatchRequest        *rcenterpb.FinishMatchRequest
}

func (f *fakeRCenterServiceClient) RegisterBattleNode(ctx context.Context, in *rcenterpb.RegisterBattleNodeRequest, opts ...grpc.CallOption) (*rcenterpb.RegisterBattleNodeResponse, error) {
	f.registerBattleNodeRequest = in
	if f.err != nil {
		return nil, f.err
	}
	return &rcenterpb.RegisterBattleNodeResponse{}, f.err
}

func (f *fakeRCenterServiceClient) ListBattleNodes(ctx context.Context, in *rcenterpb.ListBattleNodesRequest, opts ...grpc.CallOption) (*rcenterpb.ListBattleNodesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.listBattleNodesResponse, nil
}

func (f *fakeRCenterServiceClient) StartMatch(ctx context.Context, in *rcenterpb.StartMatchRequest, opts ...grpc.CallOption) (*rcenterpb.StartMatchResponse, error) {
	f.startMatchRequest = in
	if f.err != nil {
		return nil, f.err
	}
	return f.startMatchResponse, nil
}

func (f *fakeRCenterServiceClient) CancelMatch(ctx context.Context, in *rcenterpb.CancelMatchRequest, opts ...grpc.CallOption) (*rcenterpb.CancelMatchResponse, error) {
	f.cancelMatchRequest = in
	if f.err != nil {
		return nil, f.err
	}
	return &rcenterpb.CancelMatchResponse{}, nil
}

func (f *fakeRCenterServiceClient) FinishMatch(ctx context.Context, in *rcenterpb.FinishMatchRequest, opts ...grpc.CallOption) (*rcenterpb.FinishMatchResponse, error) {
	f.finishMatchRequest = in
	if f.err != nil {
		return nil, f.err
	}
	return &rcenterpb.FinishMatchResponse{}, nil
}
