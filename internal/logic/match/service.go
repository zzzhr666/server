package match

import (
	"context"
	"server/internal/rcenter"
)

// Repository 定义局外匹配服务使用的 rcenter 操作。
type Repository interface {
	StartMatch(ctx context.Context, playerID int64, weapon string) (*rcenter.MatchResult, error)
	CancelMatch(ctx context.Context, playerID int64) error
	ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
}

// Service 定义提供给逻辑服 TCP 实时处理器的匹配操作。
type Service interface {
	Start(ctx context.Context, playerID int64, weapon string) (*rcenter.MatchResult, error)
	Cancel(ctx context.Context, playerID int64) error
	Resume(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
}

// GameMatchService 在委托 rcenter 前校验局外匹配请求。
type GameMatchService struct {
	matchRepo Repository
}

// Start 通过 rcenter 将玩家入队或完成匹配。
func (g *GameMatchService) Start(ctx context.Context, playerID int64, weapon string) (*rcenter.MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, rcenter.ErrInvalidPlayerID
	}
	if weapon == "" {
		weapon = "sword"
	}
	return g.matchRepo.StartMatch(ctx, playerID, weapon)
}

// Cancel 将玩家移出 rcenter 等待队列。
func (g *GameMatchService) Cancel(ctx context.Context, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if playerID <= 0 {
		return rcenter.ErrInvalidPlayerID
	}
	return g.matchRepo.CancelMatch(ctx, playerID)
}

// Resume 返回玩家可恢复的活跃战斗分配。
func (g *GameMatchService) Resume(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, rcenter.ErrInvalidPlayerID
	}
	return g.matchRepo.ResumeMatch(ctx, playerID)
}

// NewService 使用 rcenter 仓储创建局外匹配服务。
func NewService(matchRepo Repository) *GameMatchService {
	return &GameMatchService{matchRepo: matchRepo}
}

var _ Service = (*GameMatchService)(nil)
