package match

import (
	"context"
	"server/internal/rcenter"
)

// Repository defines the rcenter operations used by the logic match service.
type Repository interface {
	StartMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
	CancelMatch(ctx context.Context, playerID int64) error
}

// Service defines matchmaking actions exposed to logic HTTP and WebSocket handlers.
type Service interface {
	Start(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
	Cancel(ctx context.Context, playerID int64) error
}

// GameMatchService validates logic requests before delegating to rcenter.
type GameMatchService struct {
	matchRepo Repository
}

// Start queues or matches a player through rcenter.
func (g *GameMatchService) Start(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, rcenter.ErrInvalidPlayerID
	}
	return g.matchRepo.StartMatch(ctx, playerID)
}

// Cancel removes a player from the rcenter waiting queue.
func (g *GameMatchService) Cancel(ctx context.Context, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if playerID <= 0 {
		return rcenter.ErrInvalidPlayerID
	}
	return g.matchRepo.CancelMatch(ctx, playerID)
}

// NewService creates a logic match service backed by a rcenter repository.
func NewService(matchRepo Repository) *GameMatchService {
	return &GameMatchService{matchRepo: matchRepo}
}

var _ Service = (*GameMatchService)(nil)
