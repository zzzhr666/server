package grpcclient

import (
	"context"
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"server/internal/rcenter/rcenterproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client adapts generated rcenter gRPC bindings to the domain contract.
type Client struct {
	client rcenterpb.RCenterServiceClient
}

// NewClient creates an rcenter client from generated gRPC bindings.
func NewClient(client rcenterpb.RCenterServiceClient) *Client {
	return &Client{client: client}
}

// StartMatch asks rcenter to queue or match one player.
func (c *Client) StartMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	res, err := c.client.StartMatch(ctx, &rcenterpb.StartMatchRequest{
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

// RegisterBattleNode registers a battle node through rcenter gRPC.
func (c *Client) RegisterBattleNode(ctx context.Context, node rcenter.BattleNode) error {
	_, err := c.client.RegisterBattleNode(ctx, &rcenterpb.RegisterBattleNodeRequest{
		Node: rcenterproto.ToProtoBattleNode(node),
	})
	return mapGRPCError(err)
}

// ListBattleNodes returns the current rcenter battle node snapshot.
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
		}
	case codes.Unavailable:
		switch st.Message() {
		case rcenter.ErrNoAvailableBattleNode.Error():
			return rcenter.ErrNoAvailableBattleNode
		}
	case codes.FailedPrecondition:
		switch st.Message() {
		case rcenter.ErrPlayerNotWaiting.Error():
			return rcenter.ErrPlayerNotWaiting
		}

	}

	return err
}
