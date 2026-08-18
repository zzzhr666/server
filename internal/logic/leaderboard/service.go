package leaderboard

import "context"

// Service 定义提供给局外协议层的排行榜查询操作。
type Service interface {
	List(ctx context.Context, input ListInput) (*Result, error)
}

// Repository 定义排行榜服务依赖的持久化查询操作。
type Repository interface {
	List(ctx context.Context, input ListInput) (*Result, error)
}

// GameLeaderboardService 校验并规范化排行榜查询。
type GameLeaderboardService struct {
	repo Repository
}

// List 校验榜单类型、地图版本和数量后查询排行榜。
func (g *GameLeaderboardService) List(ctx context.Context, input ListInput) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Limit == 0 {
		input.Limit = 20
	}
	if input.Limit < 0 || input.Limit > 100 {
		return nil, ErrInvalidQuery
	}

	switch input.Type {
	case TypeDuoClearTime, TypeSoloClearTime:
		if input.MapVersion == "" {
			return nil, ErrInvalidQuery
		}
	case TypeTotalKills:
		input.MapVersion = ""
	default:
		return nil, ErrInvalidQuery
	}
	return g.repo.List(ctx, input)
}

// NewService 使用排行榜仓储创建局外排行榜服务。
func NewService(repo Repository) *GameLeaderboardService {
	return &GameLeaderboardService{repo: repo}
}

var _ Service = (*GameLeaderboardService)(nil)
