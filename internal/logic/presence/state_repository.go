package presence

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"time"
)

// StateRepository adapts presence storage to the shared state contract.
type StateRepository struct {
	stateClient statecontract.PresenceClient
}

// NewStateRepository creates a presence repository backed by state-server.
func NewStateRepository(client statecontract.PresenceClient) *StateRepository {
	return &StateRepository{stateClient: client}
}

// SetPresence stores an online-state record through state-server.
func (s *StateRepository) SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error {
	return mapStateError(s.stateClient.SetPresence(ctx, toStatePresence(presence), ttl))
}

// GetPresence loads an online-state record through state-server.
func (s *StateRepository) GetPresence(ctx context.Context, playerID int64) (*Presence, error) {
	presence, err := s.stateClient.GetPresence(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStatePresence(presence), nil
}

// ClearPresence removes an online-state record for the owning logic-server.
func (s *StateRepository) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	return mapStateError(s.stateClient.ClearPresence(ctx, playerID, serverName))
}

func toStatePresence(p *Presence) *statecontract.Presence {
	if p == nil {
		return nil
	}
	return &statecontract.Presence{
		PlayerID:   p.PlayerID,
		ServerName: p.ServerName,
		Status:     p.Status,
		UpdatedAt:  p.UpdatedAt,
	}
}

func fromStatePresence(p *statecontract.Presence) *Presence {
	if p == nil {
		return nil
	}
	return &Presence{
		PlayerID:   p.PlayerID,
		ServerName: p.ServerName,
		Status:     p.Status,
		UpdatedAt:  p.UpdatedAt,
	}
}

func mapStateError(err error) error {
	switch {
	case errors.Is(err, statecontract.ErrPresenceNotFound):
		return ErrNotFound
	case errors.Is(err, statecontract.ErrInvalidPresence):
		return ErrInvalidPresence
	default:
		return err
	}
}
