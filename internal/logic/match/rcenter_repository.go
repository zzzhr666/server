package match

import (
	"context"
	"server/internal/rcenter"
)

type rCenterClient interface {
	StartMatch(ctx context.Context, playerID int64, weapon string, solo bool) (*rcenter.MatchResult, error)
	CancelMatch(ctx context.Context, playerID int64) error
	ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
}

// RCenterRepository 将 rcenter 客户端适配为局外匹配仓储。
type RCenterRepository struct {
	client rCenterClient
}

// NewRCenterRepository 使用 rcenter gRPC 客户端创建匹配仓储。
func NewRCenterRepository(client rCenterClient) *RCenterRepository {
	return &RCenterRepository{client: client}
}

// StartMatch 将开始匹配请求转发给 rcenter。
func (r *RCenterRepository) StartMatch(ctx context.Context, playerID int64, weapon string, solo bool) (*rcenter.MatchResult, error) {
	return r.client.StartMatch(ctx, playerID, weapon, solo)
}

// CancelMatch 将取消匹配请求转发给 rcenter。
func (r *RCenterRepository) CancelMatch(ctx context.Context, playerID int64) error {
	return r.client.CancelMatch(ctx, playerID)
}

// ResumeMatch 将活跃战斗分配查询转发给 rcenter。
func (r *RCenterRepository) ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	return r.client.ResumeMatch(ctx, playerID)
}
