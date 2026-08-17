package player

import "context"

// Service 定义玩家领域业务操作。
type Service interface {
	// Create 创建昵称非空的玩家。
	Create(ctx context.Context, input CreateInput) (*Player, error)
	// Get 按 ID 返回玩家。
	Get(ctx context.Context, id int64) (*Player, error)
	// UpdateAvatar 校验并更新玩家的预定义头像。
	UpdateAvatar(ctx context.Context, id int64, avatar string) (*Player, error)
}

// Repository 定义玩家服务依赖的持久化操作。
type Repository interface {
	// NextID 分配下一个玩家 ID。
	NextID(ctx context.Context) (int64, error)
	// Create 持久化玩家。
	Create(ctx context.Context, p *Player) error
	// Get 按 ID 读取玩家。
	Get(ctx context.Context, id int64) (*Player, error)
	// UpdateAvatar 持久化玩家的头像标识并返回更新后的玩家。
	UpdateAvatar(ctx context.Context, id int64, avatar string) (*Player, error)
}

// GamePlayerService 实现玩家领域规则。
type GamePlayerService struct {
	playersRepo Repository
}

// CreateInput 包含创建玩家时使用的档案字段。
type CreateInput struct {
	Nickname string
	Avatar   string
	Email    string
	Phone    string
}

// NewService 使用指定仓储创建玩家服务。
func NewService(repo Repository) *GamePlayerService {
	return &GamePlayerService{playersRepo: repo}
}

// Create 校验玩家昵称后创建玩家。
func (s *GamePlayerService) Create(ctx context.Context, input CreateInput) (*Player, error) {
	if input.Nickname == "" {
		return nil, ErrInvalidNickname
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := s.playersRepo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	p := &Player{
		ID:       id,
		Nickname: input.Nickname,
		Avatar:   input.Avatar,
		Email:    input.Email,
		Phone:    input.Phone,
	}
	if err := s.playersRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Get 按 ID 返回玩家。
func (s *GamePlayerService) Get(ctx context.Context, id int64) (*Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.playersRepo.Get(ctx, id)
}

// UpdateAvatar 校验头像标识后更新玩家头像。
func (s *GamePlayerService) UpdateAvatar(ctx context.Context, id int64, avatar string) (*Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resolved, err := ResolveAvatar(avatar)
	if err != nil {
		return nil, err
	}
	return s.playersRepo.UpdateAvatar(ctx, id, string(resolved))
}
