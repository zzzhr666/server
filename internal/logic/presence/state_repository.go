package presence

import (
	"context"
	"errors"
	"server/internal/contract/state"
	"time"
)

// StateRepository 将在线状态存储适配到共享状态契约。
type StateRepository struct {
	stateClient state.PresenceClient
}

// NewStateRepository 使用 state-server 创建在线状态仓储。
func NewStateRepository(client state.PresenceClient) *StateRepository {
	return &StateRepository{stateClient: client}
}

// SetPresence 通过 state-server 存储在线状态记录。
func (s *StateRepository) SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error {
	return mapStateError(s.stateClient.SetPresence(ctx, toStatePresence(presence), ttl))
}

// GetPresence 通过 state-server 读取在线状态记录。
func (s *StateRepository) GetPresence(ctx context.Context, playerID int64) (*Presence, error) {
	presence, err := s.stateClient.GetPresence(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStatePresence(presence), nil
}

// ClearPresence 删除所属 logic-server 的在线状态记录。
func (s *StateRepository) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	return mapStateError(s.stateClient.ClearPresence(ctx, playerID, serverName))
}

// RefreshPresence 通过 state-server 延长在线状态。
func (s *StateRepository) RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	return mapStateError(s.stateClient.RefreshPresence(ctx, playerID, serverName, updatedAt, ttl))
}

func toStatePresence(p *Presence) *state.Presence {
	if p == nil {
		return nil
	}
	return &state.Presence{
		PlayerID:   p.PlayerID,
		ServerName: p.ServerName,
		Status:     p.Status,
		UpdatedAt:  p.UpdatedAt,
	}
}

func fromStatePresence(p *state.Presence) *Presence {
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
	case errors.Is(err, state.ErrPresenceNotFound):
		return ErrNotFound
	case errors.Is(err, state.ErrInvalidPresence):
		return ErrInvalidPresence
	default:
		return err
	}
}
