package httpapi

import (
	"context"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/player"
	"server/internal/logic/presence"
)

// Handler owns the HTTP and WebSocket routes for a logic-server instance.
type Handler struct {
	authService        auth.Service
	serverName         string
	presenceService    presence.Service
	connections        *connManager
	friendService      friend.Service
	playerService      player.Service
	realtimeSubscriber *realtimeSubscriber
	realtimeClient     statecontract.RealtimeClient
}

// HandlerConfig wires logic services into the HTTP adapter.
type HandlerConfig struct {
	AuthService     auth.Service
	ServerName      string
	PresenceService presence.Service
	FriendService   friend.Service
	PlayerService   player.Service
	RealtimeClient  statecontract.RealtimeClient
}

// NewHandler creates an HTTP handler with logic-server services.
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
	}
}

func (h *Handler) RunRealtimeSubscriber(ctx context.Context) error {
	if h.realtimeSubscriber == nil {
		return nil
	}
	return h.realtimeSubscriber.Run(ctx)
}

// publishFriendPresenceChanged notifies online friends about player presence changes.
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

// publishFriendRemoved notifies the removed player that removerID is no longer a friend.
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
