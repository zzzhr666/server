package presence

import (
	"context"
	"time"
)

// Service 定义 HTTP 和 WebSocket 层使用的在线状态操作。
type Service interface {
	MarkOnline(ctx context.Context, playerID int64, serverName string) error
	Get(ctx context.Context, playerID int64) (*Presence, error)
	MarkOffline(ctx context.Context, playerID int64, serverName string) error
	Refresh(ctx context.Context, playerID int64, serverName string) error
}

// Repository 在状态服务中存储和清理在线记录。
type Repository interface {
	SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error
}

// GamePresenceService 在持久化前校验在线状态操作。
type GamePresenceService struct {
	presencesRepo Repository
}

// NewService 使用指定仓储创建在线状态服务。
func NewService(repo Repository) *GamePresenceService {
	return &GamePresenceService{presencesRepo: repo}
}

// MarkOffline 清除指定 logic-server 实例持有的玩家在线记录。
func (g *GamePresenceService) MarkOffline(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return ErrInvalidPresence
	}
	return g.presencesRepo.ClearPresence(ctx, playerID, serverName)
}

// MarkOnline 记录玩家已连接到指定 logic-server 实例。
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

// Get 返回玩家当前的在线记录。
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

// Refresh 为所属 logic-server 延长玩家在线状态的 TTL。
func (g *GamePresenceService) Refresh(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return ErrInvalidPresence
	}
	return g.presencesRepo.RefreshPresence(ctx, playerID, serverName, time.Now(), DefaultTTL)
}

var _ Service = (*GamePresenceService)(nil)
