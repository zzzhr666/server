package match

import (
	"context"
	"server/internal/platform/logging"
	"server/internal/rcenter"
)

// Repository 定义局外匹配服务使用的 rcenter 操作。
type Repository interface {
	// StartMatch 向 rcenter 发起单人对局或双人匹配。
	StartMatch(ctx context.Context, playerID int64, nickname, hero string, solo bool) (*rcenter.MatchResult, error)
	// CancelMatch 从 rcenter 等待队列取消匹配。
	CancelMatch(ctx context.Context, playerID int64) error
	// ResumeMatch 从 rcenter 读取玩家的活跃战斗分配。
	ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
}

// Service 定义提供给逻辑服 TCP 实时处理器的匹配操作。
type Service interface {
	// Start 发起单人对局或双人匹配。
	Start(ctx context.Context, playerID int64, nickname, hero string, solo bool) (*rcenter.MatchResult, error)
	// Cancel 取消等待中的匹配。
	Cancel(ctx context.Context, playerID int64) error
	// Resume 返回玩家可恢复的活跃战斗分配。
	Resume(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
}

// GameMatchService 在委托 rcenter 前校验局外匹配请求。
type GameMatchService struct {
	matchRepo Repository
}

// Start 通过 rcenter 发起单人对局或双人匹配。
func (g *GameMatchService) Start(ctx context.Context, playerID int64, nickname, hero string, solo bool) (*rcenter.MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, rcenter.ErrInvalidPlayerID
	}
	if hero == "" {
		hero = "fire"
	}
	result, err := g.matchRepo.StartMatch(ctx, playerID, nickname, hero, solo)
	if err != nil {
		logging.Warn("match start failed player_id=%d: %v", playerID, err)
		return nil, err
	}
	if result != nil {
		logging.Info("match start accepted player_id=%d status=%s", playerID, result.Status)
	} else {
		logging.Warn("match start returned empty result player_id=%d", playerID)
	}
	return result, nil
}

// Cancel 将玩家移出 rcenter 等待队列。
func (g *GameMatchService) Cancel(ctx context.Context, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if playerID <= 0 {
		return rcenter.ErrInvalidPlayerID
	}
	if err := g.matchRepo.CancelMatch(ctx, playerID); err != nil {
		logging.Warn("match cancel failed player_id=%d: %v", playerID, err)
		return err
	}
	logging.Info("match canceled player_id=%d", playerID)
	return nil
}

// Resume 返回玩家可恢复的活跃战斗分配。
func (g *GameMatchService) Resume(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, rcenter.ErrInvalidPlayerID
	}
	result, err := g.matchRepo.ResumeMatch(ctx, playerID)
	if err != nil {
		logging.Debug("match resume unavailable player_id=%d: %v", playerID, err)
		return nil, err
	}
	if result != nil {
		logging.Info("match resumed player_id=%d room=%s", playerID, result.RoomName)
	} else {
		logging.Warn("match resume returned empty result player_id=%d", playerID)
	}
	return result, nil
}

// NewService 使用 rcenter 仓储创建局外匹配服务。
func NewService(matchRepo Repository) *GameMatchService {
	return &GameMatchService{matchRepo: matchRepo}
}

var _ Service = (*GameMatchService)(nil)
