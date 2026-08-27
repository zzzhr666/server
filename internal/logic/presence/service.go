package presence

import (
	"context"
	"server/internal/platform/logging"
	"time"
)

// Service 定义 HTTP 和 TCP 层使用的在线状态操作。
type Service interface {
	// MarkOnline 将玩家标记为连接到指定 logic-server。
	MarkOnline(ctx context.Context, playerID int64, serverName string) error
	// Get 返回玩家当前在线状态。
	Get(ctx context.Context, playerID int64) (*Presence, error)
	// MarkOffline 仅在服务所有权匹配时清除在线状态。
	MarkOffline(ctx context.Context, playerID int64, serverName string) error
	// Refresh 续期指定 logic-server 持有的在线状态。
	Refresh(ctx context.Context, playerID int64, serverName string) error
}

// Repository 在状态服务中存储和清理在线记录。
type Repository interface {
	// SetPresence 持久化带 TTL 的在线状态。
	SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error
	// GetPresence 读取玩家在线状态。
	GetPresence(ctx context.Context, playerID int64) (*Presence, error)
	// ClearPresence 仅在服务所有权匹配时删除在线状态。
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	// RefreshPresence 仅在服务所有权匹配时刷新在线状态 TTL。
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
	if err := g.presencesRepo.ClearPresence(ctx, playerID, serverName); err != nil {
		logging.Error("mark player offline failed player_id=%d server=%s: %v", playerID, serverName, err)
		return err
	}
	logging.Debug("player offline player_id=%d server=%s", playerID, serverName)
	return nil
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
	if err := g.presencesRepo.SetPresence(ctx, presence, DefaultTTL); err != nil {
		logging.Error("mark player online failed player_id=%d server=%s: %v", playerID, serverName, err)
		return err
	}
	logging.Debug("player online player_id=%d server=%s", playerID, serverName)
	return nil
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
	if err := g.presencesRepo.RefreshPresence(ctx, playerID, serverName, time.Now(), DefaultTTL); err != nil {
		logging.Warn("refresh player presence failed player_id=%d server=%s: %v", playerID, serverName, err)
		return err
	}
	logging.Trace("player presence refreshed player_id=%d server=%s", playerID, serverName)
	return nil
}

var _ Service = (*GamePresenceService)(nil)
