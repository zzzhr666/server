package presence

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"time"
)

// StateRepository 将在线状态存储适配到共享状态契约。
type StateRepository struct {
	stateClient statecontract.PresenceClient
}

// NewStateRepository 使用 state-server 创建在线状态仓储。
func NewStateRepository(client statecontract.PresenceClient) *StateRepository {
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
