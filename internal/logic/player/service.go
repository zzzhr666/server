package player

import "context"

// Service defines player business operations.
type Service interface {
	// Create creates a player with a non-empty nickname.
	Create(ctx context.Context, input CreateInput) (*Player, error)
	// Get returns a player by ID.
	Get(ctx context.Context, id int64) (*Player, error)
}

// Repository defines player persistence operations used by the service layer.
type Repository interface {
	// NextID allocates the next player ID.
	NextID(ctx context.Context) (int64, error)
	// Create persists a player.
	Create(ctx context.Context, p *Player) error
	// Get loads a player by ID.
	Get(ctx context.Context, id int64) (*Player, error)
}

// GamePlayerService implements player business rules.
type GamePlayerService struct {
	playersRepo Repository
}

// CreateInput contains player profile fields used during player creation.
type CreateInput struct {
	Nickname string
	Avatar   string
	Email    string
	Phone    string
}

// NewService creates a player service with the given repository.
func NewService(repo Repository) *GamePlayerService {
	return &GamePlayerService{playersRepo: repo}
}

// Create creates a player after validating the player nickname.
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

// Get returns a player by ID.
func (s *GamePlayerService) Get(ctx context.Context, id int64) (*Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.playersRepo.Get(ctx, id)
}
