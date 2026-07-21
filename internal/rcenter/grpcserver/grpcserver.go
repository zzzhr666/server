package grpcserver

import (
	"context"
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"server/internal/rcenter/rcenterproto"
)

// Server exposes the rcenter domain service through generated gRPC bindings.
type Server struct {
	rcenterpb.UnimplementedRCenterServiceServer
	center *rcenter.GameCenterService
}

// NewServer creates a gRPC rcenter server adapter.
func NewServer(center *rcenter.GameCenterService) *Server {
	return &Server{
		center: center,
	}
}

// RegisterBattleNode handles battle node registration requests.
func (s *Server) RegisterBattleNode(ctx context.Context, req *rcenterpb.RegisterBattleNodeRequest) (*rcenterpb.RegisterBattleNodeResponse, error) {
	if err := s.center.RegisterBattleNode(ctx, rcenterproto.FromProtoBattleNode(req.GetNode())); err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.RegisterBattleNodeResponse{}, nil
}

// ListBattleNodes handles battle node listing requests.
func (s *Server) ListBattleNodes(ctx context.Context, req *rcenterpb.ListBattleNodesRequest) (*rcenterpb.ListBattleNodesResponse, error) {
	currentNodes := s.center.ListBattleNodes()
	respNodes := make([]*rcenterpb.BattleNode, 0, len(currentNodes))
	for _, node := range currentNodes {
		respNodes = append(respNodes, rcenterproto.ToProtoBattleNode(node))
	}
	return &rcenterpb.ListBattleNodesResponse{
		Nodes: respNodes,
	}, nil

}

// StartMatch handles player matchmaking requests.
func (s *Server) StartMatch(ctx context.Context, req *rcenterpb.StartMatchRequest) (*rcenterpb.StartMatchResponse, error) {
	res, err := s.center.StartMatch(ctx, req.GetPlayerId())
	if err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.StartMatchResponse{Result: rcenterproto.ToProtoMatchResult(res)}, nil

}

// CancelMatch handles queue cancellation requests.
func (s *Server) CancelMatch(ctx context.Context, req *rcenterpb.CancelMatchRequest) (*rcenterpb.CancelMatchResponse, error) {
	err := s.center.CancelMatch(ctx, req.GetPlayerId())
	if err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.CancelMatchResponse{}, nil
}
