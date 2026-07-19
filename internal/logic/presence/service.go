package presence

import (
	"context"
	"time"
)

// Service defines online-state operations used by the HTTP/WebSocket layer.
type Service interface {
	MarkOnline(ctx context.Context, playerID int64, serverName string) error
	Get(ctx context.Context, playerID int64) (*Presence, error)
	MarkOffline(ctx context.Context, playerID int64, serverName string) error
}

// Repository stores and clears presence records in the state service.
type Repository interface {
	SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
}

// GamePresenceService validates online-state operations before storage.
type GamePresenceService struct {
	presencesRepo Repository
}

// NewService creates a presence service backed by repo.
func NewService(repo Repository) *GamePresenceService {
	return &GamePresenceService{presencesRepo: repo}
}

// MarkOffline clears a player's presence for the given logic-server instance.
func (g *GamePresenceService) MarkOffline(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return ErrInvalidPresence
	}
	return g.presencesRepo.ClearPresence(ctx, playerID, serverName)
}

// MarkOnline records that a player is connected to a logic-server instance.
func (g *GamePresenceService) MarkOnline(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return ErrInvalidPresence
	}
	presence := &Presence{
		PlayerID:   playerID,
		ServerName: serverName,
		Status:     StatusOnline,
		UpdatedAt:  time.Now(),
	}
	return g.presencesRepo.SetPresence(ctx, presence, DefaultTTL)
}

// Get returns the current presence record for a player.
func (g *GamePresenceService) Get(ctx context.Context, playerID int64) (*Presence, error) {
	if playerID <= 0 {
		return nil, ErrInvalidPresence
	}
	presence, err := g.presencesRepo.GetPresence(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return presence, nil
}

var _ Service = (*GamePresenceService)(nil)
