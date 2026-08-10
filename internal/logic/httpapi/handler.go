package httpapi

import (
	"context"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/player"
	"server/internal/logic/presence"
)

// Handler 持有一个 logic-server 实例的 HTTP 路由。
type Handler struct {
	authService     auth.Service
	serverName      string
	presenceService presence.Service
	friendService   friend.Service
	playerService   player.Service
	realtimeClient  statecontract.RealtimeClient
	growthService   growth.Service
}

// HandlerConfig 将逻辑服务注入 HTTP 适配器。
type HandlerConfig struct {
	AuthService     auth.Service
	ServerName      string
	PresenceService presence.Service
	FriendService   friend.Service
	PlayerService   player.Service
	RealtimeClient  statecontract.RealtimeClient
	GrowthService   growth.Service
}

// NewHandler 使用 logic-server 服务创建 HTTP 处理器。
func NewHandler(handlerConfig HandlerConfig) *Handler {
	return &Handler{
		authService:     handlerConfig.AuthService,
		serverName:      handlerConfig.ServerName,
		presenceService: handlerConfig.PresenceService,
		friendService:   handlerConfig.FriendService,
		playerService:   handlerConfig.PlayerService,
		realtimeClient:  handlerConfig.RealtimeClient,
		growthService:   handlerConfig.GrowthService,
	}
}

// publishFriendPresenceChanged 通知在线好友玩家在线状态的变化。
func (h *Handler) publishFriendPresenceChanged(ctx context.Context, playerID int64, online bool, status string) {
	if h.realtimeClient == nil {
		return
	}

	friendIDs, err := h.friendService.ListFriendIDs(ctx, playerID)
	if err != nil {
		return
	}
	for _, friendID := range friendIDs {
		friendPresence, err := h.presenceService.Get(ctx, friendID)
		if err != nil {
			continue
		}
		event := &statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventFriendPresenceChanged,
			TargetPlayerID: friendID,
			ActorPlayerID:  playerID,
			Online:         online,
			Status:         status,
		}
		_ = h.realtimeClient.PublishRealtimeToServer(ctx, friendPresence.ServerName, event)
	}
}

// publishFriendRemoved 通知被删除玩家，removerID 不再是其好友。
func (h *Handler) publishFriendRemoved(ctx context.Context, removedPlayerID, removerID int64) {
	if h.realtimeClient == nil {
		return
	}
	targetPresence, err := h.presenceService.Get(ctx, removedPlayerID)
	if err != nil {
		return
	}
	event := &statecontract.RealtimeEvent{
		Type:           statecontract.RealtimeEventFriendRemoved,
		TargetPlayerID: removedPlayerID,
		ActorPlayerID:  removerID,
	}
	_ = h.realtimeClient.PublishRealtimeToServer(ctx, targetPresence.ServerName, event)
}

func (h *Handler) publishRealtimeToOnlinePlayer(ctx context.Context, targetPlayerID, actorPlayerID int64, eventType string) {
	if h.realtimeClient == nil {
		return
	}
	targetPresence, err := h.presenceService.Get(ctx, targetPlayerID)
	if err != nil {
		return
	}
	event := &statecontract.RealtimeEvent{
		Type:           eventType,
		TargetPlayerID: targetPlayerID,
		ActorPlayerID:  actorPlayerID,
	}
	_ = h.realtimeClient.PublishRealtimeToServer(ctx, targetPresence.ServerName, event)
}

func (h *Handler) publishFriendRequestReceived(ctx context.Context, toPlayerID, fromPlayerID int64) {
	h.publishRealtimeToOnlinePlayer(ctx, toPlayerID, fromPlayerID, statecontract.RealtimeEventFriendRequestReceived)
}

func (h *Handler) publishFriendRequestHandled(ctx context.Context, fromPlayerID, handledByPlayerID int64) {
	h.publishRealtimeToOnlinePlayer(ctx, fromPlayerID, handledByPlayerID, statecontract.RealtimeEventFriendRequestHandled)
}
