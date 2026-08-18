package realtime

import (
	"context"
	"errors"
	"server/internal/contract/realtimepb"
	"server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/chat"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/leaderboard"
	"server/internal/logic/match"
	"server/internal/logic/player"
	"server/internal/logic/presence"
	"server/internal/platform/logging"
	"server/internal/rcenter"
	"time"
)

const (
	defaultAuthenticationTimeout = 5 * time.Second
	defaultIdleTimeout           = 60 * time.Second
)

// Handler 处理局外原生 TCP 实时协议。
type Handler struct {
	auth                  auth.Service
	presence              presence.Service
	match                 match.Service
	friend                friend.Service
	player                player.Service
	growth                growth.Service
	chat                  chat.Service
	leaderboard           leaderboard.Service
	serverName            string
	connManager           *connectionManager
	realtimeClient        state.RealtimeClient
	subscriber            *subscriber
	metrics               *Metrics
	authenticationTimeout time.Duration
	idleTimeout           time.Duration
}

// HandlerConfig 定义 Handler 所需的服务依赖。
type HandlerConfig struct {
	AuthService           auth.Service
	PresenceService       presence.Service
	MatchService          match.Service
	FriendService         friend.Service
	PlayerService         player.Service
	GrowthService         growth.Service
	ChatService           chat.Service
	LeaderboardService    leaderboard.Service
	ServerName            string
	RealtimeClient        state.RealtimeClient
	Metrics               *Metrics
	AuthenticationTimeout time.Duration
	IdleTimeout           time.Duration
}

// NewHandler 使用服务依赖创建 TCP 协议处理器。
func NewHandler(config HandlerConfig) *Handler {
	handler := &Handler{
		auth:                  config.AuthService,
		presence:              config.PresenceService,
		match:                 config.MatchService,
		friend:                config.FriendService,
		player:                config.PlayerService,
		growth:                config.GrowthService,
		chat:                  config.ChatService,
		leaderboard:           config.LeaderboardService,
		serverName:            config.ServerName,
		connManager:           newConnectionManager(),
		metrics:               config.Metrics,
		authenticationTimeout: config.AuthenticationTimeout,
		idleTimeout:           config.IdleTimeout,
	}
	if config.RealtimeClient != nil {
		handler.realtimeClient = config.RealtimeClient
		handler.subscriber = newSubscriber(config.ServerName, config.RealtimeClient, handler.connManager, handler.metrics)
	}
	return handler
}

func (h *Handler) serveSession(ctx context.Context, session *session) {
	authSession, requestID, ok := h.handleAuthenticate(ctx, session)
	if !ok {
		logging.Debug("realtime session authentication failed")
		return
	}
	logging.Info("realtime session authenticated player_id=%d", authSession.PlayerID)
	connInfo, ok := h.handleConnectionReady(ctx, session, authSession, requestID)
	if !ok {
		return
	}

	defer func() {
		if !h.connManager.Remove(authSession.PlayerID, connInfo.id) {
			return
		}
		if h.metrics != nil {
			h.metrics.Connections.Dec()
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := h.presence.MarkOffline(cleanupCtx, authSession.PlayerID, h.serverName); err != nil {
			logging.Error("realtime offline cleanup failed player_id=%d: %v", authSession.PlayerID, err)
			return
		}
		logging.Info("realtime session disconnected player_id=%d", authSession.PlayerID)
		h.publishFriendPresenceChanged(cleanupCtx, authSession.PlayerID, false, presence.StatusOffline)
	}()
	if !h.handleAuthenticated(ctx, session, authSession.PlayerID, requestID) {
		return
	}
	idleTimeout := h.idleTimeout
	if idleTimeout <= 0 {
		idleTimeout = defaultIdleTimeout
	}

	for {
		if err := session.setReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			logging.Error("realtime session setReadDeadline failed: %v", err)
			return
		}
		envelope, err := session.Read()
		if err != nil {
			return
		}
		if !h.handleEnvelope(ctx, session, authSession, connInfo.id, envelope) {
			return
		}
	}
}

func (h *Handler) handleAuthenticate(ctx context.Context, session *session) (*auth.Session, uint64, bool) {
	timeout := h.authenticationTimeout
	if timeout <= 0 {
		timeout = defaultAuthenticationTimeout
	}

	if err := session.setReadDeadline(time.Now().Add(timeout)); err != nil {
		logging.Error("session set read deadline failed: %v", err)
		return nil, 0, false
	}
	envelope, readErr := session.Read()
	clearErr := session.setReadDeadline(time.Time{})
	if readErr != nil {
		logging.Warn("realtime authenticate read failed: %v", readErr)
		return nil, 0, false
	}
	if clearErr != nil {
		logging.Warn("clear read timeout failed: %v", clearErr)
		return nil, 0, false
	}
	request := envelope.GetAuthenticate()
	if request == nil {
		logging.Warn("realtime authenticate request missing")
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INVALID_ARGUMENT, "invalid argument")
		return nil, 0, false
	}

	authSession, err := h.auth.GetSession(ctx, request.GetToken())
	if errors.Is(err, auth.ErrSessionNotFound) {
		logging.Warn("realtime authentication rejected reason=invalid_session")
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_UNAUTHENTICATED, "invalid session")
		return nil, 0, false
	}
	if err != nil {
		logging.Error("realtime authentication lookup failed: %v", err)
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "internal server error")
		return nil, 0, false
	}
	if authSession == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "internal server error")
		return nil, 0, false
	}
	return authSession, envelope.GetRequestId(), true
}

func (h *Handler) handleConnectionReady(ctx context.Context, session *session, authSession *auth.Session, requestID uint64) (connectionInfo, bool) {
	h.replaceExistingConnection(ctx, authSession.PlayerID)
	if err := h.presence.MarkOnline(ctx, authSession.PlayerID, h.serverName); err != nil {
		logging.Error("realtime online setup failed player_id=%d: %v", authSession.PlayerID, err)
		_ = writeError(session, requestID, realtimepb.ErrorCode_INTERNAL, "internal server error")
		return connectionInfo{}, false
	}

	connInfo, oldConn := h.connManager.Add(authSession.PlayerID, session)
	if h.metrics != nil {
		h.metrics.Connections.Inc()
	}
	h.publishFriendPresenceChanged(ctx, authSession.PlayerID, true, presence.StatusOnline)
	if oldConn != nil {
		logging.Warn("realtime connection replaced player_id=%d", authSession.PlayerID)
		_ = oldConn.session.Write(&realtimepb.ServerEnvelope{
			RequestId: 0,
			Payload: &realtimepb.ServerEnvelope_ConnectionReplaced{
				ConnectionReplaced: &realtimepb.ConnectionReplaced{},
			},
		})
		_ = oldConn.session.Close()
	}

	return connInfo, true
}

func (h *Handler) handleAuthenticated(ctx context.Context, session *session, playerID int64, requestID uint64) bool {
	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ServerEnvelope_Authenticated{
			Authenticated: &realtimepb.Authenticated{PlayerId: playerID},
		},
	}); err != nil {
		logging.Error("realtime authenticated response failed player_id=%d: %v", playerID, err)
		return false
	}
	return h.handleAutomaticMatchResume(ctx, session, playerID)
}

func (h *Handler) handleAutomaticMatchResume(ctx context.Context, session *session, playerID int64) bool {
	if h.match == nil {
		return true
	}
	resumeResult, err := h.match.Resume(ctx, playerID)
	switch {
	case errors.Is(err, rcenter.ErrActiveMatchNotFound):
		return true
	case err != nil:
		logging.Error("automatic match resume failed player_id=%d: %v", playerID, err)
		return true
	case resumeResult == nil || resumeResult.Status == rcenter.MatchStatusUnexpected:
		return false
	default:
		return session.Write(matchResultEnvelope(proactivePushRequestID, resumeResult)) == nil
	}
}

func (h *Handler) handleEnvelope(ctx context.Context, session *session, authSession *auth.Session, connID connectionID, envelope *realtimepb.ClientEnvelope) (handled bool) {
	defer func() {
		h.metrics.observeRequest(realtimeRequestType(envelope), handled)
	}()
	playerID := authSession.PlayerID
	switch {
	case envelope.GetHeartbeat() != nil:
		return h.handleHeartbeat(ctx, session, playerID, connID, envelope)
	case envelope.GetMatchStart() != nil:
		return h.handleMatchStart(ctx, session, playerID, envelope)
	case envelope.GetMatchCancel() != nil:
		return h.handleMatchCancel(ctx, session, playerID, envelope)
	case envelope.GetMatchResume() != nil:
		return h.handleMatchResume(ctx, session, playerID, envelope)
	case envelope.GetPlayerGet() != nil:
		return h.handleGetPlayer(ctx, session, playerID, envelope)
	case envelope.GetGrowthGet() != nil:
		return h.handleGetGrowth(ctx, session, playerID, envelope)
	case envelope.GetGrowthUpgrade() != nil:
		return h.handleUpgradeGrowth(ctx, session, playerID, envelope)
	case envelope.GetFriendRequestSend() != nil:
		return h.handleSendFriendRequest(ctx, session, playerID, envelope)
	case envelope.GetFriendRequestListIncoming() != nil:
		return h.handleListIncomingFriendRequests(ctx, session, playerID, envelope)
	case envelope.GetFriendRequestListOutgoing() != nil:
		return h.handleListOutgoingFriendRequests(ctx, session, playerID, envelope)
	case envelope.GetFriendRequestAccept() != nil:
		return h.handleAcceptFriendRequest(ctx, session, playerID, envelope)
	case envelope.GetFriendRequestReject() != nil:
		return h.handleRejectFriendRequest(ctx, session, playerID, envelope)
	case envelope.GetFriendList() != nil:
		return h.handleListFriends(ctx, session, playerID, envelope)
	case envelope.GetFriendDelete() != nil:
		return h.handleDeleteFriend(ctx, session, playerID, envelope)
	case envelope.GetLogout() != nil:
		return h.handleLogout(ctx, session, authSession, envelope)
	case envelope.GetChatWorldSend() != nil:
		return h.handleSendWorldChat(ctx, session, playerID, envelope)
	case envelope.GetChatDirectSend() != nil:
		return h.handleSendDirectChat(ctx, session, playerID, envelope)
	case envelope.GetChatWorldList() != nil:
		return h.handleListWorldChat(ctx, session, playerID, envelope)
	case envelope.GetChatDirectList() != nil:
		return h.handleListDirectChat(ctx, session, playerID, envelope)
	case envelope.GetPlayerAvatarUpdate() != nil:
		return h.handleUpdatePlayerAvatar(ctx, session, playerID, envelope)
	case envelope.GetLeaderboardList() != nil:
		return h.handleListLeaderboard(ctx, session, envelope)
	default:
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INVALID_ARGUMENT, "unsupported message") == nil
	}
}

func realtimeRequestType(envelope *realtimepb.ClientEnvelope) string {
	if envelope == nil {
		return "unknown"
	}
	switch {
	case envelope.GetHeartbeat() != nil:
		return "heartbeat"
	case envelope.GetMatchStart() != nil, envelope.GetMatchCancel() != nil, envelope.GetMatchResume() != nil:
		return "match"
	case envelope.GetPlayerGet() != nil, envelope.GetPlayerAvatarUpdate() != nil:
		return "player"
	case envelope.GetGrowthGet() != nil, envelope.GetGrowthUpgrade() != nil:
		return "growth"
	case envelope.GetFriendRequestSend() != nil, envelope.GetFriendRequestListIncoming() != nil,
		envelope.GetFriendRequestListOutgoing() != nil, envelope.GetFriendRequestAccept() != nil,
		envelope.GetFriendRequestReject() != nil, envelope.GetFriendList() != nil, envelope.GetFriendDelete() != nil:
		return "friend"
	case envelope.GetChatWorldSend() != nil, envelope.GetChatDirectSend() != nil,
		envelope.GetChatWorldList() != nil, envelope.GetChatDirectList() != nil:
		return "chat"
	case envelope.GetLeaderboardList() != nil:
		return "leaderboard"
	case envelope.GetLogout() != nil:
		return "logout"
	default:
		return "unknown"
	}
}

func (h *Handler) handleUpgradeGrowth(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.growth == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "growth service unavailable")
		return false
	}
	upgradeType := toUpgradeTypeName(envelope.GetGrowthUpgrade().GetType())
	result, err := h.growth.Upgrade(ctx, playerID, upgradeType)
	if err != nil {
		return writeError(session, envelope.GetRequestId(), growthErrorCode(err), err.Error()) == nil
	}
	if result == nil || result.Growth == nil {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected growth result") == nil
	}
	options, err := h.growth.UpgradeOptions(result.Growth)
	if err != nil {
		return writeError(session, envelope.GetRequestId(), growthErrorCode(err), err.Error()) == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_GrowthUpgradeResult{
			GrowthUpgradeResult: toProtoUpgradeGrowthResponse(result, options),
		},
	}) == nil
}

func (h *Handler) handleSendFriendRequest(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	toPlayerID := envelope.GetFriendRequestSend().GetToPlayerId()
	if err := h.friend.SendRequest(ctx, playerID, toPlayerID); err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	actorNickname := ""
	targetNickname := ""
	if h.player != nil {
		if actor, err := h.player.Get(ctx, playerID); err == nil && actor != nil {
			actorNickname = actor.Nickname
		}
		if target, err := h.player.Get(ctx, toPlayerID); err == nil && target != nil {
			targetNickname = target.Nickname
		}
	}
	h.pushRealtimeEvent(ctx, &state.RealtimeEvent{
		Type:                state.RealtimeEventFriendRequestReceived,
		TargetPlayerID:      toPlayerID,
		ActorPlayerID:       playerID,
		ActorPlayerNickname: actorNickname,
	})
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_FriendRequestSent{
			FriendRequestSent: &realtimepb.FriendRequestSentResponse{
				ToPlayerId: toPlayerID,
				ToNickname: targetNickname,
			},
		},
	}) == nil
}

func (h *Handler) handleListIncomingFriendRequests(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	requests, err := h.friend.ListIncomingRequests(ctx, playerID)
	if err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_FriendRequests{
			FriendRequests: &realtimepb.FriendRequestResponse{
				Requests: h.toProtoFriendRequestsWithNicknames(ctx, requests),
			},
		},
	}) == nil

}

func (h *Handler) handleListOutgoingFriendRequests(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	requests, err := h.friend.ListOutgoingRequests(ctx, playerID)
	if err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_FriendRequests{
			FriendRequests: &realtimepb.FriendRequestResponse{
				Requests: h.toProtoFriendRequestsWithNicknames(ctx, requests),
			},
		},
	}) == nil
}

func (h *Handler) toProtoFriendRequestsWithNicknames(ctx context.Context, requests []*friend.Request) []*realtimepb.FriendRequest {
	result := make([]*realtimepb.FriendRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		fromNickname, toNickname := "", ""
		if h.player != nil {
			if from, err := h.player.Get(ctx, request.FromPlayerID); err == nil && from != nil {
				fromNickname = from.Nickname
			}
			if to, err := h.player.Get(ctx, request.ToPlayerID); err == nil && to != nil {
				toNickname = to.Nickname
			}
		}
		result = append(result, toProtoFriendRequestWithNicknames(request, fromNickname, toNickname))
	}
	return result
}

func (h *Handler) handleAcceptFriendRequest(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	fromPlayerID := envelope.GetFriendRequestAccept().GetFromPlayerId()
	if err := h.friend.AcceptRequest(ctx, fromPlayerID, playerID); err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	h.pushRealtimeEvent(ctx, &state.RealtimeEvent{
		Type:           state.RealtimeEventFriendRequestHandled,
		TargetPlayerID: fromPlayerID,
		ActorPlayerID:  playerID,
	})
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_FriendRequestHandledAck{
			FriendRequestHandledAck: &realtimepb.FriendRequestHandledResponse{},
		},
	}) == nil
}

func (h *Handler) handleRejectFriendRequest(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	fromPlayerID := envelope.GetFriendRequestReject().GetFromPlayerId()
	if err := h.friend.RejectRequest(ctx, fromPlayerID, playerID); err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	h.pushRealtimeEvent(ctx, &state.RealtimeEvent{
		Type:           state.RealtimeEventFriendRequestHandled,
		TargetPlayerID: fromPlayerID,
		ActorPlayerID:  playerID,
	})
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_FriendRequestHandledAck{
			FriendRequestHandledAck: &realtimepb.FriendRequestHandledResponse{},
		},
	}) == nil
}

func (h *Handler) handleListFriends(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	friendIDs, err := h.friend.ListFriendIDs(ctx, playerID)
	if err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	friendSummaries := make([]*realtimepb.FriendSummary, 0, len(friendIDs))
	for _, friendID := range friendIDs {
		friendPlayer, err := h.player.Get(ctx, friendID)
		if err != nil {
			return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "internal server error") == nil
		}
		if friendPlayer == nil {
			return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected player result") == nil
		}
		friendPresence, err := h.presence.Get(ctx, friendID)
		if errors.Is(err, presence.ErrNotFound) {
			friendPresence = nil
		} else if err != nil {
			return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "internal server error") == nil
		}
		friendSummaries = append(friendSummaries, toProtoFriendSummary(friendPlayer, friendPresence))
	}

	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_Friends{
			Friends: &realtimepb.FriendListResponse{
				Friends: friendSummaries,
			},
		},
	}) == nil
}

func (h *Handler) handleDeleteFriend(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	targetID := envelope.GetFriendDelete().GetFriendPlayerId()
	if err := h.friend.DeleteFriend(ctx, playerID, targetID); err != nil {
		return writeFriendError(session, envelope.GetRequestId(), err) == nil
	}
	h.pushRealtimeEvent(ctx, &state.RealtimeEvent{
		Type:           state.RealtimeEventFriendRemoved,
		TargetPlayerID: targetID,
		ActorPlayerID:  playerID,
	})
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_FriendDeleted{
			FriendDeleted: &realtimepb.FriendDeletedResponse{},
		},
	}) == nil
}

func (h *Handler) handleLogout(ctx context.Context, session *session, authSession *auth.Session, envelope *realtimepb.ClientEnvelope) bool {
	err := h.auth.Logout(ctx, authSession.Token)
	switch {
	case errors.Is(err, auth.ErrSessionNotFound):
		_ = writeError(
			session,
			envelope.GetRequestId(),
			realtimepb.ErrorCode_UNAUTHENTICATED,
			"invalid session",
		)
		return false

	case err != nil:
		return writeError(
			session,
			envelope.GetRequestId(),
			realtimepb.ErrorCode_INTERNAL,
			"internal server error",
		) == nil
	}

	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_LogoutAck{
			LogoutAck: &realtimepb.LogoutResponse{},
		},
	}); err != nil {
		return false
	}

	return false
}

func (h *Handler) handleGetGrowth(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.growth == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "growth service unavailable")
		return false
	}
	growthInfo, err := h.growth.Get(ctx, playerID)
	if err != nil {
		return writeError(session, envelope.GetRequestId(), growthErrorCode(err), err.Error()) == nil
	}
	if growthInfo == nil {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected growth result") == nil
	}
	upgradeOption, err := h.growth.UpgradeOptions(growthInfo)
	if err != nil {
		return writeError(session, envelope.GetRequestId(), growthErrorCode(err), err.Error()) == nil
	}
	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_Growth{
			Growth: &realtimepb.GrowthResponse{
				Growth: toProtoGrowth(growthInfo, upgradeOption),
			},
		},
	}); err != nil {
		return false
	}
	return true
}

func (h *Handler) handleGetPlayer(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.player == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "player service unavailable")
		return false
	}
	playerInfo, err := h.player.Get(ctx, playerID)
	if err != nil {
		return writeError(session, envelope.GetRequestId(), playerErrorCode(err), err.Error()) == nil
	}
	if playerInfo == nil {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected player result") == nil
	}
	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_Player{
			Player: &realtimepb.PlayerResponse{
				Player: toProtoPlayer(playerInfo),
			},
		},
	}); err != nil {
		return false
	}
	return true

}

func (h *Handler) handleHeartbeat(ctx context.Context, session *session, playerID int64, connID connectionID, envelope *realtimepb.ClientEnvelope) bool {
	if err := h.presence.Refresh(ctx, playerID, h.serverName); err != nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "presence refresh failed")
		return false
	}
	if !h.connManager.Touch(playerID, connID, time.Now()) {
		return false
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload:   &realtimepb.ServerEnvelope_HeartbeatAck{HeartbeatAck: &realtimepb.HeartbeatAck{}},
	}) == nil
}

func (h *Handler) handleMatchStart(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	weapon := envelope.GetMatchStart().GetWeapon()
	if weapon == "" {
		weapon = "sword"
	}
	if !isValidWeapon(weapon) {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INVALID_ARGUMENT, "invalid weapon") == nil
	}
	if h.match == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "match service unavailable")
		return false
	}

	matchResult, err := h.match.Start(ctx, playerID, weapon, envelope.GetMatchStart().GetSolo())
	if err != nil {
		return writeError(session, envelope.GetRequestId(), matchErrorCode(err), err.Error()) == nil
	}
	if matchResult == nil || matchResult.Status == rcenter.MatchStatusUnexpected {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected match result") == nil
	}
	if err := session.Write(matchResultEnvelope(envelope.GetRequestId(), matchResult)); err != nil {
		return false
	}
	h.pushMatchResultToPlayers(ctx, playerID, matchResult)
	return true
}

func (h *Handler) handleMatchCancel(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.match == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "match service unavailable")
		return false
	}
	if err := h.match.Cancel(ctx, playerID); err != nil {
		return writeError(session, envelope.GetRequestId(), matchErrorCode(err), err.Error()) == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload:   &realtimepb.ServerEnvelope_MatchCanceled{MatchCanceled: &realtimepb.MatchCanceled{}},
	}) == nil
}

func (h *Handler) handleMatchResume(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.match == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "match service unavailable")
		return false
	}
	matchResult, err := h.match.Resume(ctx, playerID)
	if err != nil {
		return writeError(session, envelope.GetRequestId(), matchErrorCode(err), err.Error()) == nil
	}
	if matchResult == nil || matchResult.Status == rcenter.MatchStatusUnexpected {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected match result") == nil
	}
	return session.Write(matchResultEnvelope(envelope.GetRequestId(), matchResult)) == nil
}

func (h *Handler) handleSendWorldChat(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.chat == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "chat service unavailable")
		return false
	}
	sendChatWorld := envelope.GetChatWorldSend()
	senderNickname := ""
	if h.player != nil {
		if sender, getErr := h.player.Get(ctx, playerID); getErr == nil && sender != nil {
			senderNickname = sender.Nickname
		}
	}

	message, err := h.chat.SendWorldMessage(ctx, chat.SendWorldMessageInput{
		SenderID:         playerID,
		Content:          sendChatWorld.GetContent(),
		ClientMessageKey: sendChatWorld.GetClientMessageKey(),
		SenderNickname:   senderNickname,
	})
	if err != nil {
		return writeError(session, envelope.GetRequestId(), chatErrorCode(err), err.Error()) == nil
	}

	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_ChatSent{
			ChatSent: &realtimepb.ChatSendResponse{
				Message: toProtoMessage(message),
			},
		},
	}); err != nil {
		return false
	}
	delivery := &state.RealtimeDelivery{
		Route: state.RealtimeRoute{
			Type: state.RealtimeRouteBroadcast,
		},
		Event: &state.RealtimeEvent{
			Type:          state.RealtimeEventChatMessage,
			ActorPlayerID: message.SenderID,
			ChatMessage:   toStateChatMessage(message),
		},
	}
	if h.realtimeClient != nil {
		if err := h.realtimeClient.PublishRealtime(ctx, delivery); err != nil {
			logging.Error("publish world chat realtime failed sender_id=%d: %v", playerID, err)
		}
	}
	logging.Info("world chat sent sender_id=%d message_key=%s", playerID, message.MessageKey)
	return true
}

func (h *Handler) handleSendDirectChat(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.chat == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "chat service unavailable")
		return false
	}
	sendChatDirect := envelope.GetChatDirectSend()
	senderNickname := ""
	if h.player != nil {
		if sender, getErr := h.player.Get(ctx, playerID); getErr == nil && sender != nil {
			senderNickname = sender.Nickname
		}
	}
	message, err := h.chat.SendDirectMessage(ctx, chat.SendDirectMessageInput{
		SenderID:         playerID,
		ReceiverID:       sendChatDirect.GetReceiverId(),
		Content:          sendChatDirect.GetContent(),
		ClientMessageKey: sendChatDirect.GetClientMessageKey(),
		SenderNickname:   senderNickname,
	})
	if err != nil {
		return writeError(session, envelope.GetRequestId(), chatErrorCode(err), err.Error()) == nil
	}
	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_ChatSent{
			ChatSent: &realtimepb.ChatSendResponse{
				Message: toProtoMessage(message),
			},
		},
	}); err != nil {
		return false
	}
	h.pushRealtimeEvent(ctx, &state.RealtimeEvent{
		Type:           state.RealtimeEventChatMessage,
		TargetPlayerID: message.ReceiverID,
		ActorPlayerID:  message.SenderID,
		ChatMessage:    toStateChatMessage(message),
	})
	logging.Info("direct chat sent sender_id=%d receiver_id=%d message_key=%s", playerID, message.ReceiverID, message.MessageKey)
	return true
}

func (h *Handler) handleListWorldChat(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.chat == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "chat service unavailable")
		return false
	}
	listWorldChat := envelope.GetChatWorldList()
	messages, err := h.chat.ListWorldMessages(ctx, chat.ListWorldMessagesInput{
		PlayerID:         playerID,
		Limit:            listWorldChat.GetLimit(),
		BeforeMessageKey: listWorldChat.GetBeforeMessageKey(),
	})
	if err != nil {
		return writeError(session, envelope.GetRequestId(), chatErrorCode(err), err.Error()) == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_ChatMessages{
			ChatMessages: &realtimepb.ChatMessagesResponse{
				Messages: toProtoMessages(messages),
			},
		},
	}) == nil
}

func (h *Handler) handleListDirectChat(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.chat == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "chat service unavailable")
		return false
	}
	listDirectChat := envelope.GetChatDirectList()
	messages, err := h.chat.ListDirectMessages(ctx, chat.ListDirectMessagesInput{
		PlayerID:         playerID,
		FriendID:         listDirectChat.GetFriendId(),
		Limit:            listDirectChat.GetLimit(),
		BeforeMessageKey: listDirectChat.GetBeforeMessageKey(),
	})
	if err != nil {
		return writeError(session, envelope.GetRequestId(), chatErrorCode(err), err.Error()) == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_ChatMessages{
			ChatMessages: &realtimepb.ChatMessagesResponse{
				Messages: toProtoMessages(messages),
			},
		},
	}) == nil
}

func (h *Handler) handleUpdatePlayerAvatar(ctx context.Context, session *session, playerID int64, envelope *realtimepb.ClientEnvelope) bool {
	if h.player == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "player service unavailable")
		return false
	}
	updateAvatar := envelope.GetPlayerAvatarUpdate()
	updated, err := h.player.UpdateAvatar(ctx, playerID, updateAvatar.GetAvatar())
	if err != nil {
		return writeError(session, envelope.GetRequestId(), playerErrorCode(err), err.Error()) == nil
	}
	if updated == nil {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected player result") == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_Player{
			Player: &realtimepb.PlayerResponse{
				Player: toProtoPlayer(updated),
			},
		},
	}) == nil
}

func (h *Handler) handleListLeaderboard(ctx context.Context, session *session, envelope *realtimepb.ClientEnvelope) bool {
	if h.leaderboard == nil {
		_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "leaderboard service unavailable")
		return false
	}
	res, err := h.leaderboard.List(ctx, toLeaderboardListInput(envelope.GetLeaderboardList()))
	if err != nil {
		code := leaderboardErrorCode(err)
		message := "internal server error"
		if code == realtimepb.ErrorCode_INVALID_ARGUMENT {
			message = err.Error()
		}
		return writeError(session, envelope.GetRequestId(), code, message) == nil
	}
	if res == nil {
		return writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected leaderboard result") == nil
	}
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: envelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_Leaderboard{
			Leaderboard: toProtoLeaderboardResult(res),
		},
	}) == nil
}

// RunRealtimeSubscriber 启动当前 logic-server 的实时事件订阅。
func (h *Handler) RunRealtimeSubscriber(ctx context.Context) error {
	if h.subscriber == nil {
		return nil
	}
	return h.subscriber.Run(ctx)
}

func (h *Handler) replaceExistingConnection(ctx context.Context, playerID int64) {
	if h.realtimeClient == nil {
		return
	}
	existingPresence, err := h.presence.Get(ctx, playerID)
	if err != nil || existingPresence == nil || existingPresence.ServerName == "" || existingPresence.ServerName == h.serverName {
		return
	}
	event := &state.RealtimeEvent{
		Type:           state.RealtimeEventConnectionReplaced,
		TargetPlayerID: playerID,
		ActorPlayerID:  playerID,
	}
	_ = h.realtimeClient.PublishRealtime(ctx, newServerDelivery(existingPresence.ServerName, event))
}

func writeError(session *session, id uint64, code realtimepb.ErrorCode, msg string) error {
	return session.Write(&realtimepb.ServerEnvelope{
		RequestId: id,
		Payload: &realtimepb.ServerEnvelope_Error{
			Error: &realtimepb.ProtocolError{
				Code:    code,
				Message: msg,
			},
		},
	})
}

func chatErrorCode(err error) realtimepb.ErrorCode {
	switch {
	case errors.Is(err, chat.ErrInvalidPlayerID), errors.Is(err, chat.ErrInvalidMessage), errors.Is(err, chat.ErrInvalidChannel):
		return realtimepb.ErrorCode_INVALID_ARGUMENT
	case errors.Is(err, chat.ErrMessageNotFound):
		return realtimepb.ErrorCode_NOT_FOUND
	case errors.Is(err, chat.ErrMessageExists), errors.Is(err, chat.ErrFriendRequired):
		return realtimepb.ErrorCode_CONFLICT
	default:
		return realtimepb.ErrorCode_INTERNAL
	}
}

func leaderboardErrorCode(err error) realtimepb.ErrorCode {
	if errors.Is(err, leaderboard.ErrInvalidQuery) {
		return realtimepb.ErrorCode_INVALID_ARGUMENT
	}
	return realtimepb.ErrorCode_INTERNAL
}

func (h *Handler) publishFriendPresenceChanged(ctx context.Context, playerID int64, online bool, status string) {
	if h.friend == nil {
		return
	}
	friendIDs, err := h.friend.ListFriendIDs(ctx, playerID)
	if err != nil {
		return
	}
	for _, friendID := range friendIDs {
		event := &state.RealtimeEvent{
			Type:           state.RealtimeEventFriendPresenceChanged,
			TargetPlayerID: friendID,
			ActorPlayerID:  playerID,
			Online:         online,
			Status:         status,
		}
		h.pushRealtimeEvent(ctx, event)
	}
}

// pushRealtimeEvent 优先向本机连接推送事件，未命中时再转发到玩家所在的 logic-server。
func (h *Handler) pushRealtimeEvent(ctx context.Context, event *state.RealtimeEvent) {
	if event == nil || ctx.Err() != nil {
		return
	}
	serverEnvelope, ok := toProtoEnvelope(*event)
	if !ok {
		return
	}
	if h.connManager.Send(event.TargetPlayerID, serverEnvelope) {
		h.metrics.observeDelivery("player", true)
		return
	}
	if h.realtimeClient == nil || h.presence == nil {
		h.metrics.observeDelivery("player", false)
		return
	}
	targetPresence, err := h.presence.Get(ctx, event.TargetPlayerID)
	if err != nil {
		h.metrics.observeDelivery("player", false)
		return
	}
	if err := h.realtimeClient.PublishRealtime(ctx, newServerDelivery(targetPresence.ServerName, event)); err != nil {
		h.metrics.observeDelivery("player", false)
	}
}

func (h *Handler) pushMatchResultToPlayers(ctx context.Context, currentPlayerID int64, result *rcenter.MatchResult) {
	if result == nil || result.Status != rcenter.MatchStatusMatched {
		return
	}

	for _, playerID := range result.PlayerIDs {
		if playerID == currentPlayerID {
			continue
		}
		event := &state.RealtimeEvent{
			Type:           state.RealtimeEventMatchResult,
			TargetPlayerID: playerID,
			ActorPlayerID:  currentPlayerID,
			MatchStatus:    string(result.Status),
			RoomName:       result.RoomName,
			MatchToken:     result.Token,
			BattleNodeName: result.BattleNodeName,
			BattleUDPAddr:  result.BattleUDPAddr,
			MatchPlayerIDs: result.PlayerIDs,
		}
		serverEnvelope, ok := toProtoEnvelope(*event)
		if !ok {
			continue
		}
		if h.connManager.Send(playerID, serverEnvelope) {
			continue
		}
		friendPresence, err := h.presence.Get(ctx, playerID)
		if err != nil {
			continue
		}
		if h.realtimeClient == nil {
			continue
		}
		_ = h.realtimeClient.PublishRealtime(ctx, newServerDelivery(friendPresence.ServerName, event))
	}
}

func newServerDelivery(serverName string, event *state.RealtimeEvent) *state.RealtimeDelivery {
	return &state.RealtimeDelivery{
		Route: state.RealtimeRoute{
			Type:       state.RealtimeRouteServer,
			ServerName: serverName,
		},
		Event: event,
	}
}

func matchErrorCode(err error) realtimepb.ErrorCode {
	switch {
	case errors.Is(err, rcenter.ErrInvalidPlayerID):
		return realtimepb.ErrorCode_INVALID_ARGUMENT
	case errors.Is(err, rcenter.ErrPlayerInGame), errors.Is(err, rcenter.ErrPlayerNotWaiting):
		return realtimepb.ErrorCode_CONFLICT
	case errors.Is(err, rcenter.ErrActiveMatchNotFound):
		return realtimepb.ErrorCode_NOT_FOUND

	default:
		return realtimepb.ErrorCode_INTERNAL
	}
}

func growthErrorCode(err error) realtimepb.ErrorCode {
	switch {
	case errors.Is(err, growth.ErrInvalidPlayerID),
		errors.Is(err, growth.ErrInvalidUpgradeType),
		errors.Is(err, growth.ErrInvalidGrowthLevel):
		return realtimepb.ErrorCode_INVALID_ARGUMENT
	case errors.Is(err, growth.ErrGrowthNotFound):
		return realtimepb.ErrorCode_NOT_FOUND
	case errors.Is(err, growth.ErrInsufficientCoins), errors.Is(err, growth.ErrMaxLevelReached):
		return realtimepb.ErrorCode_CONFLICT
	default:
		return realtimepb.ErrorCode_INTERNAL
	}
}

func playerErrorCode(err error) realtimepb.ErrorCode {
	switch {
	case errors.Is(err, player.ErrInvalidAvatar):
		return realtimepb.ErrorCode_INVALID_ARGUMENT
	case errors.Is(err, player.ErrNotFound):
		return realtimepb.ErrorCode_NOT_FOUND
	default:
		return realtimepb.ErrorCode_INTERNAL
	}
}

func isValidWeapon(weapon string) bool {
	if weapon == "bow" || weapon == "axe" || weapon == "dagger" || weapon == "sword" {
		return true
	}
	return false
}

func matchResultEnvelope(requestID uint64, result *rcenter.MatchResult) *realtimepb.ServerEnvelope {
	return &realtimepb.ServerEnvelope{
		RequestId: requestID,
		Payload: &realtimepb.ServerEnvelope_MatchResult{
			MatchResult: &realtimepb.MatchResult{
				Status:         string(result.Status),
				RoomName:       result.RoomName,
				Token:          result.Token,
				BattleNodeName: result.BattleNodeName,
				BattleUdpAddr:  result.BattleUDPAddr,
			},
		},
	}
}

func writeFriendError(session *session, requestID uint64, err error) error {
	switch {
	case errors.Is(err, friend.ErrInvalidPlayerID) || errors.Is(err, friend.ErrInvalidRequest):
		return writeError(session, requestID, realtimepb.ErrorCode_INVALID_ARGUMENT, err.Error())
	case errors.Is(err, friend.ErrRequestNotFound) || errors.Is(err, friend.ErrNotFound):
		return writeError(session, requestID, realtimepb.ErrorCode_NOT_FOUND, err.Error())
	case errors.Is(err, friend.ErrAlreadyExists) || errors.Is(err, friend.ErrRequestExists):
		return writeError(session, requestID, realtimepb.ErrorCode_CONFLICT, err.Error())
	}
	return writeError(session, requestID, realtimepb.ErrorCode_INTERNAL, "internal server error")
}
