package player

import "context"

type Service interface {
	Create(ctx context.Context, name string) (*Player, error)
	Get(ctx context.Context, id int64) (*Player, error)
}

type Repository interface {
	NextID(ctx context.Context) (int64, error)
	Create(ctx context.Context, p *Player) error
	Get(ctx context.Context, id int64) (*Player, error)
	Exists(ctx context.Context, id int64) (bool, error)
}

type PlayerService struct {
	repo Repository
}

func NewService(repo Repository) *PlayerService {
	return &PlayerService{repo: repo}
}

func (s *PlayerService) Create(ctx context.Context, name string) (*Player, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := s.repo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	p := &Player{ID: id, Name: name}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PlayerService) Get(ctx context.Context, id int64) (*Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}
