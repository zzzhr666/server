package player

import "context"

// Service defines player business operations.
type Service interface {
	// Create creates a player with a non-empty name.
	Create(ctx context.Context, input CreateInput) (*Player, error)
	// Get returns a player by ID.
	Get(ctx context.Context, id int64) (*Player, error)
	// UpdateProfile updates the provided profile fields for an existing player.
	UpdateProfile(ctx context.Context, id int64, input UpdateProfileInput) (*Player, error)
}

// Repository defines player persistence operations used by the service layer.
type Repository interface {
	// NextID allocates the next player ID.
	NextID(ctx context.Context) (int64, error)
	// Create persists a player.
	Create(ctx context.Context, p *Player) error
	// Get loads a player by ID.
	Get(ctx context.Context, id int64) (*Player, error)
	// Exists reports whether a player exists.
	Exists(ctx context.Context, id int64) (bool, error)
	// UpdateProfile persists profile changes for an existing player.
	UpdateProfile(ctx context.Context, p *Player) error
}

// GamePlayerService implements player business rules.
type GamePlayerService struct {
	playersRepo Repository
}

// CreateInput contains player profile fields used during player creation.
type CreateInput struct {
	Name   string
	Avatar string
	Email  string
	Phone  string
}

// UpdateProfileInput contains optional player profile fields to update.
type UpdateProfileInput struct {
	Name   *string
	Avatar *string
	Email  *string
	Phone  *string
}

// NewService creates a player service with the given repository.
func NewService(repo Repository) *GamePlayerService {
	return &GamePlayerService{playersRepo: repo}
}

// Create creates a player after validating the player name.
func (s *GamePlayerService) Create(ctx context.Context, input CreateInput) (*Player, error) {
	if input.Name == "" {
		return nil, ErrInvalidName
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := s.playersRepo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	p := &Player{
		ID:     id,
		Name:   input.Name,
		Avatar: input.Avatar,
		Email:  input.Email,
		Phone:  input.Phone,
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

// UpdateProfile applies partial profile changes after validating the player name.
func (s *GamePlayerService) UpdateProfile(ctx context.Context, id int64, input UpdateProfileInput) (*Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := s.playersRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		if *input.Name == "" {
			return nil, ErrInvalidName
		}
		p.Name = *input.Name
	}
	if input.Avatar != nil {
		p.Avatar = *input.Avatar
	}
	if input.Email != nil {
		p.Email = *input.Email
	}
	if input.Phone != nil {
		p.Phone = *input.Phone
	}
	if err := s.playersRepo.UpdateProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
