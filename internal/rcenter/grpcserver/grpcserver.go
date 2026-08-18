package grpcserver

import (
	"context"
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"server/internal/rcenter/rcenterproto"
)

// Server 通过生成的 gRPC 绑定暴露 rcenter 领域服务。
type Server struct {
	rcenterpb.UnimplementedRCenterServiceServer
	center *rcenter.GameCenterService
}

// NewServer 创建 gRPC rcenter 服务适配器。
func NewServer(center *rcenter.GameCenterService) *Server {
	return &Server{
		center: center,
	}
}

// RegisterBattleNode 处理战斗节点注册请求。
func (s *Server) RegisterBattleNode(ctx context.Context, req *rcenterpb.RegisterBattleNodeRequest) (*rcenterpb.RegisterBattleNodeResponse, error) {
	if err := s.center.RegisterBattleNode(ctx, rcenterproto.FromProtoBattleNode(req.GetNode())); err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.RegisterBattleNodeResponse{}, nil
}

// ListBattleNodes 处理战斗节点列表请求。
func (s *Server) ListBattleNodes(context.Context, *rcenterpb.ListBattleNodesRequest) (*rcenterpb.ListBattleNodesResponse, error) {
	currentNodes := s.center.ListBattleNodes()
	respNodes := make([]*rcenterpb.BattleNode, 0, len(currentNodes))
	for _, node := range currentNodes {
		respNodes = append(respNodes, rcenterproto.ToProtoBattleNode(node))
	}
	return &rcenterpb.ListBattleNodesResponse{
		Nodes: respNodes,
	}, nil

}

// StartMatch 处理玩家发起单人对局或双人匹配的请求。
func (s *Server) StartMatch(ctx context.Context, req *rcenterpb.StartMatchRequest) (*rcenterpb.StartMatchResponse, error) {
	res, err := s.center.StartMatch(ctx, req.GetPlayerId(), req.GetWeapon(), req.GetSolo())
	if err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.StartMatchResponse{Result: rcenterproto.ToProtoMatchResult(res)}, nil

}

// ResumeMatch 返回玩家的活跃战斗分配。
func (s *Server) ResumeMatch(ctx context.Context, request *rcenterpb.ResumeMatchRequest) (*rcenterpb.ResumeMatchResponse, error) {
	res, err := s.center.ResumeMatch(ctx, request.GetPlayerId())
	if err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.ResumeMatchResponse{
		Result: rcenterproto.ToProtoMatchResult(res),
	}, nil
}

// CancelMatch 处理队列取消请求。
func (s *Server) CancelMatch(ctx context.Context, req *rcenterpb.CancelMatchRequest) (*rcenterpb.CancelMatchResponse, error) {
	err := s.center.CancelMatch(ctx, req.GetPlayerId())
	if err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.CancelMatchResponse{}, nil
}

// FinishMatch 处理战斗服发来的战斗完成通知。
func (s *Server) FinishMatch(ctx context.Context, request *rcenterpb.FinishMatchRequest) (*rcenterpb.FinishMatchResponse, error) {
	input := rcenter.FinishMatchInput{
		RoomName:         request.GetRoomName(),
		PlayerIDs:        request.GetPlayerIds(),
		Reason:           request.GetReason(),
		CombatDurationMS: request.GetCombatDurationMs(),
	}
	for _, stat := range request.GetPlayerStats() {
		input.PlayerStats = append(input.PlayerStats, rcenterproto.FromProtoPlayerBattleStats(stat))
	}

	if err := s.center.FinishMatch(ctx, input); err != nil {
		return nil, mapRCenterError(err)
	}
	return &rcenterpb.FinishMatchResponse{}, nil
}
