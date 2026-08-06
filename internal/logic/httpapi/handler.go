package httpapi

import (
	"context"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/match"
	"server/internal/logic/player"
	"server/internal/logic/presence"
)

// Handler 持有一个 logic-server 实例的 HTTP 和 WebSocket 路由。
type Handler struct {
	authService        auth.Service
	serverName         string
	presenceService    presence.Service
	connections        *connManager
	friendService      friend.Service
	playerService      player.Service
	realtimeSubscriber *realtimeSubscriber
	realtimeClient     statecontract.RealtimeClient
	matchService       match.Service
	growthService      growth.Service
}

// HandlerConfig 将逻辑服务注入 HTTP 适配器。
type HandlerConfig struct {
	AuthService     auth.Service
	ServerName      string
	PresenceService presence.Service
	FriendService   friend.Service
	PlayerService   player.Service
	RealtimeClient  statecontract.RealtimeClient
	MatchService    match.Service
	GrowthService   growth.Service
}

// NewHandler 使用 logic-server 服务创建 HTTP 处理器。
func NewHandler(handlerConfig HandlerConfig) *Handler {
	connections := newConnManager()
	var subscriber *realtimeSubscriber
	if handlerConfig.RealtimeClient != nil {
		subscriber = newRealtimeSubscriber(handlerConfig.ServerName, handlerConfig.RealtimeClient, newLocalRealtimePusher(connections))
	}
	return &Handler{
		authService:        handlerConfig.AuthService,
		serverName:         handlerConfig.ServerName,
		presenceService:    handlerConfig.PresenceService,
		connections:        connections,
		friendService:      handlerConfig.FriendService,
		playerService:      handlerConfig.PlayerService,
		realtimeSubscriber: subscriber,
		realtimeClient:     handlerConfig.RealtimeClient,
		matchService:       handlerConfig.MatchService,
		growthService:      handlerConfig.GrowthService,
	}
}

func (h *Handler) RunRealtimeSubscriber(ctx context.Context) error {
	if h.realtimeSubscriber == nil {
		return nil
	}
	return h.realtimeSubscriber.Run(ctx)
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

func (h *Handler) replaceExistingConnection(ctx context.Context, playerID int64) {
	if h.realtimeClient == nil {
		return
	}
	// presence 记录的是上一次持有该玩家的 logic-server。向该实例发布事件而不是
	// 本地直接 Close，才能处理负载均衡后同一账号落在不同进程的情况。
	existingPresence, err := h.presenceService.Get(ctx, playerID)
	if err != nil {
		return
	}
	event := &statecontract.RealtimeEvent{
		Type:           statecontract.RealtimeEventConnectionReplaced,
		TargetPlayerID: playerID,
		ActorPlayerID:  playerID,
	}
	_ = h.realtimeClient.PublishRealtimeToServer(ctx, existingPresence.ServerName, event)
}
