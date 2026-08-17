package grpcclient

import (
	"context"
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"server/internal/rcenter/rcenterproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client 将生成的 rcenter gRPC 绑定适配到领域契约。
type Client struct {
	client rcenterpb.RCenterServiceClient
}

// NewClient 使用生成的 gRPC 绑定创建 rcenter 客户端。
func NewClient(client rcenterpb.RCenterServiceClient) *Client {
	return &Client{client: client}
}

// StartMatch 请求 rcenter 发起单人对局或双人匹配。
func (c *Client) StartMatch(ctx context.Context, playerID int64, weapon string, solo bool) (*rcenter.MatchResult, error) {
	res, err := c.client.StartMatch(ctx, &rcenterpb.StartMatchRequest{
		PlayerId: playerID,
		Weapon:   weapon,
		Solo:     solo,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return rcenterproto.FromProtoMatchResult(res.Result), nil
}

// ResumeMatch retrieves a player's active battle assignment from rcenter.
func (c *Client) ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	res, err := c.client.ResumeMatch(ctx, &rcenterpb.ResumeMatchRequest{
		PlayerId: playerID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return rcenterproto.FromProtoMatchResult(res.Result), nil
}

// CancelMatch asks rcenter to remove one player from the waiting queue.
func (c *Client) CancelMatch(ctx context.Context, playerID int64) error {
	_, err := c.client.CancelMatch(ctx, &rcenterpb.CancelMatchRequest{
		PlayerId: playerID,
	})
	return mapGRPCError(err)
}

// FinishMatch releases matched players through rcenter gRPC.
func (c *Client) FinishMatch(ctx context.Context, input rcenter.FinishMatchInput) error {
	req := &rcenterpb.FinishMatchRequest{
		PlayerIds: input.PlayerIDs,
		Reason:    input.Reason,
		RoomName:  input.RoomName,
	}
	for _, stat := range input.PlayerStats {
		req.PlayerStats = append(req.PlayerStats, rcenterproto.ToProtoPlayerBattleStats(stat))
	}
	_, err := c.client.FinishMatch(ctx, req)

	return mapGRPCError(err)
}

// RegisterBattleNode registers a battle node through rcenter gRPC.
func (c *Client) RegisterBattleNode(ctx context.Context, node rcenter.BattleNode) error {
	_, err := c.client.RegisterBattleNode(ctx, &rcenterpb.RegisterBattleNodeRequest{
		Node: rcenterproto.ToProtoBattleNode(node),
	})
	return mapGRPCError(err)
}

// ListBattleNodes 返回当前 rcenter 战斗节点快照。
func (c *Client) ListBattleNodes(ctx context.Context) ([]rcenter.BattleNode, error) {
	res, err := c.client.ListBattleNodes(ctx, &rcenterpb.ListBattleNodesRequest{})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	nodes := make([]rcenter.BattleNode, 0, len(res.GetNodes()))
	for _, node := range res.GetNodes() {
		nodes = append(nodes, rcenterproto.FromProtoBattleNode(node))
	}
	return nodes, nil
}

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}
	st := status.Convert(err)
	switch st.Code() {
	case codes.InvalidArgument:
		switch st.Message() {
		case rcenter.ErrInvalidBattleNode.Error():
			return rcenter.ErrInvalidBattleNode
		case rcenter.ErrInvalidPlayerID.Error():
			return rcenter.ErrInvalidPlayerID
		case rcenter.ErrInvalidRoomName.Error():
			return rcenter.ErrInvalidRoomName
		case rcenter.ErrInvalidBattleStats.Error():
			return rcenter.ErrInvalidBattleStats
		}
	case codes.Unavailable:
		switch st.Message() {
		case rcenter.ErrNoAvailableBattleNode.Error():
			return rcenter.ErrNoAvailableBattleNode
		case rcenter.ErrUnavailableCoinClient.Error():
			return rcenter.ErrUnavailableCoinClient
		case rcenter.ErrUnavailableGrowthClient.Error():
			return rcenter.ErrUnavailableGrowthClient
		}
	case codes.FailedPrecondition:
		switch st.Message() {
		case rcenter.ErrPlayerNotWaiting.Error():
			return rcenter.ErrPlayerNotWaiting
		case rcenter.ErrPlayerInGame.Error():
			return rcenter.ErrPlayerInGame
		}
	case codes.NotFound:
		switch st.Message() {
		case rcenter.ErrActiveMatchNotFound.Error():
			return rcenter.ErrActiveMatchNotFound
		}

	}

	return err
}
