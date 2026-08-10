package realtime

import (
	"context"
	"errors"
	"server/internal/contract/realtimepb"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/match"
	"server/internal/logic/presence"
	"server/internal/rcenter"
	"time"
)

// Handler 处理局外原生 TCP 实时协议。
type Handler struct {
	auth           auth.Service
	presence       presence.Service
	match          match.Service
	friend         friend.Service
	serverName     string
	connManager    *connectionManager
	realtimeClient statecontract.RealtimeClient
	subscriber     *subscriber
}

// HandlerConfig 定义 Handler 所需的服务依赖。
type HandlerConfig struct {
	AuthService     auth.Service
	PresenceService presence.Service
	MatchService    match.Service
	FriendService   friend.Service
	ServerName      string
	RealtimeClient  statecontract.RealtimeClient
}

// NewHandler 使用服务依赖创建 TCP 协议处理器。
func NewHandler(config HandlerConfig) *Handler {
	handler := &Handler{
		auth:        config.AuthService,
		presence:    config.PresenceService,
		match:       config.MatchService,
		friend:      config.FriendService,
		serverName:  config.ServerName,
		connManager: newConnectionManager(),
	}
	if config.RealtimeClient != nil {
		handler.realtimeClient = config.RealtimeClient
		handler.subscriber = newSubscriber(config.ServerName, config.RealtimeClient, handler.connManager)
	}
	return handler
}

func (h *Handler) serveSession(ctx context.Context, session *session) {
	clientEnvelope, err := session.Read()
	if err != nil {
		return
	}
	authenticateRequest := clientEnvelope.GetAuthenticate()
	if authenticateRequest == nil {
		_ = writeError(session, clientEnvelope.GetRequestId(), realtimepb.ErrorCode_INVALID_ARGUMENT, "invalid argument")
		return
	}
	authSession, err := h.auth.GetSession(ctx, authenticateRequest.GetToken())
	if errors.Is(err, auth.ErrSessionNotFound) {
		_ = writeError(session, clientEnvelope.GetRequestId(), realtimepb.ErrorCode_UNAUTHENTICATED, "invalid session")
		return
	} else if err != nil {
		_ = writeError(session, clientEnvelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "internal server error")
		return
	}

	h.replaceExistingConnection(ctx, authSession.PlayerID)

	if err := h.presence.MarkOnline(ctx, authSession.PlayerID, h.serverName); err != nil {
		_ = writeError(session, clientEnvelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "internal server error")
		return
	}

	connInfo, oldConn := h.connManager.Add(authSession.PlayerID, session)
	h.publishFriendPresenceChanged(ctx, authSession.PlayerID, true, presence.StatusOnline)

	if oldConn != nil {
		_ = oldConn.session.Write(&realtimepb.ServerEnvelope{
			RequestId: 0,
			Payload: &realtimepb.ServerEnvelope_ConnectionReplaced{
				ConnectionReplaced: &realtimepb.ConnectionReplaced{},
			},
		})
		_ = oldConn.session.Close()
	}

	defer func() {
		if !h.connManager.Remove(authSession.PlayerID, connInfo.id) {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := h.presence.MarkOffline(cleanupCtx, authSession.PlayerID, h.serverName); err != nil {
			return
		}
		h.publishFriendPresenceChanged(cleanupCtx, authSession.PlayerID, false, presence.StatusOffline)
	}()
	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: clientEnvelope.GetRequestId(),
		Payload: &realtimepb.ServerEnvelope_Authenticated{
			Authenticated: &realtimepb.Authenticated{
				PlayerId: authSession.PlayerID,
			},
		},
	}); err != nil {
		return
	}

	if h.match != nil {
		resumeResult, err := h.match.Resume(ctx, authSession.PlayerID)
		switch {
		case errors.Is(err, rcenter.ErrActiveMatchNotFound):
			// 没有可恢复对局时保持连接，等待客户端的新匹配请求。
		case err != nil:
			// 自动恢复不是客户端请求，暂不发送错误，由后续显式恢复重试。
		case resumeResult == nil || resumeResult.Status == rcenter.MatchStatusUnexpected:
			return
		case session.Write(matchResultEnvelope(proactivePushRequestID, resumeResult)) != nil:
			return
		}
	}

	for {
		envelope, err := session.Read()
		if err != nil {
			return
		}
		switch {
		case envelope.GetHeartbeat() != nil:
			err := h.presence.Refresh(ctx, authSession.PlayerID, h.serverName)
			if err != nil {
				_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "presence refresh failed")
				return
			}
			if !h.connManager.Touch(authSession.PlayerID, connInfo.id, time.Now()) {
				return
			}
			if err := session.Write(&realtimepb.ServerEnvelope{
				RequestId: envelope.GetRequestId(),
				Payload: &realtimepb.ServerEnvelope_HeartbeatAck{
					HeartbeatAck: &realtimepb.HeartbeatAck{},
				},
			}); err != nil {
				return
			}

		case envelope.GetMatchStart() != nil:
			startReq := envelope.GetMatchStart()
			weapon := startReq.GetWeapon()
			if weapon == "" {
				weapon = "sword"
			}
			if !isValidWeapon(weapon) {
				if err := writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INVALID_ARGUMENT, "invalid weapon"); err != nil {
					return
				}
				continue
			}
			if h.match == nil {
				_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "match service unavailable")
				return
			}
			matchResult, err := h.match.Start(ctx, authSession.PlayerID, weapon)
			if err != nil {
				if err := writeError(session, envelope.GetRequestId(), matchErrorCode(err), err.Error()); err != nil {
					return
				}
				continue
			}
			if matchResult == nil || matchResult.Status == rcenter.MatchStatusUnexpected {
				if err := writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected match result"); err != nil {
					return
				}
				continue
			}
			if err := session.Write(matchResultEnvelope(envelope.GetRequestId(), matchResult)); err != nil {
				return
			}
			h.pushMatchResultToPlayers(ctx, authSession.PlayerID, matchResult)

		case envelope.GetMatchCancel() != nil:
			if h.match == nil {
				_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "match service unavailable")
				return
			}
			err := h.match.Cancel(ctx, authSession.PlayerID)
			if err != nil {
				if err := writeError(session, envelope.GetRequestId(), matchErrorCode(err), err.Error()); err != nil {
					return
				}
				continue
			}
			if err := session.Write(&realtimepb.ServerEnvelope{
				RequestId: envelope.GetRequestId(),
				Payload: &realtimepb.ServerEnvelope_MatchCanceled{
					MatchCanceled: &realtimepb.MatchCanceled{},
				},
			}); err != nil {
				return
			}
		case envelope.GetMatchResume() != nil:
			if h.match == nil {
				_ = writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "match service unavailable")
				return
			}
			matchResult, err := h.match.Resume(ctx, authSession.PlayerID)
			if err != nil {
				if err := writeError(session, envelope.GetRequestId(), matchErrorCode(err), err.Error()); err != nil {
					return
				}
				continue
			}
			if matchResult == nil || matchResult.Status == rcenter.MatchStatusUnexpected {
				if err := writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INTERNAL, "unexpected match result"); err != nil {
					return
				}
				continue
			}
			if err := session.Write(matchResultEnvelope(envelope.GetRequestId(), matchResult)); err != nil {
				return
			}

		default:
			if err := writeError(session, envelope.GetRequestId(), realtimepb.ErrorCode_INVALID_ARGUMENT, "unsupported message"); err != nil {
				return
			}

		}
	}
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

func (h *Handler) publishFriendPresenceChanged(ctx context.Context, playerID int64, online bool, status string) {
	if h.realtimeClient == nil || h.friend == nil {
		return
	}
	friendIDs, err := h.friend.ListFriendIDs(ctx, playerID)
	if err != nil {
		return
	}
	for _, friendID := range friendIDs {
		friendPresence, err := h.presence.Get(ctx, friendID)
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

func (h *Handler) pushMatchResultToPlayers(ctx context.Context, currentPlayerID int64, result *rcenter.MatchResult) {
	if result == nil || result.Status != rcenter.MatchStatusMatched {
		return
	}

	for _, playerID := range result.PlayerIDs {
		if playerID == currentPlayerID {
			continue
		}
		event := &statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventMatchResult,
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
		_ = h.realtimeClient.PublishRealtimeToServer(ctx, friendPresence.ServerName, event)
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
