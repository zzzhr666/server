package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	statecontract "server/internal/contract/state"
	"server/internal/logic/presence"
	"server/internal/rcenter"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const websocketReadTimeout = 90 * time.Second

// handleWebSocket 建立玩家局外实时连接，并协调在线状态、旧连接替换和活跃对局恢复。
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing token"})
		return
	}
	session, err := h.authService.GetSession(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid session"})
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}

	defer func() {
		_ = conn.CloseNow()
	}()

	// 先通知旧连接所属 logic-server 自行关闭，再注册当前连接。在线状态最终由
	// connection ID 保护，避免旧连接的 defer 把新连接标记为离线。
	h.replaceExistingConnection(r.Context(), session.PlayerID)
	if err := h.presenceService.MarkOnline(r.Context(), session.PlayerID, h.serverName); err != nil {
		if closeErr := conn.Close(websocket.StatusInternalError, "presence failed"); closeErr != nil {
			log.Printf("close websocket failed: player_id = %d err = %v", session.PlayerID, closeErr)
		}
		return
	}
	connInfo := h.connections.Add(session.PlayerID, conn)

	h.publishFriendPresenceChanged(r.Context(), session.PlayerID, true, presence.StatusOnline)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// 只有仍是当前连接的协程才能清理 presence。新连接已经覆盖记录时，
		// Remove 返回 false，旧连接无需再发布离线事件。
		if !h.connections.Remove(session.PlayerID, connInfo.id) {
			return
		}
		if err := h.presenceService.MarkOffline(ctx, session.PlayerID, h.serverName); err != nil {
			log.Printf("mark offline failed: player_id = %d server_name = %s,err = %v", session.PlayerID, h.serverName, err)
			return
		}
		h.publishFriendPresenceChanged(ctx, session.PlayerID, false, presence.StatusOffline)
	}()
	// 活跃对局由 rcenter 持有，与 WebSocket 生命周期独立。连接建立即尝试恢复，
	// 使从战斗返回大厅或切换 logic-server 的玩家能继续使用原房间令牌。
	if err := h.sendResumedMatch(context.Background(), conn, session.PlayerID); err != nil && !errors.Is(err, rcenter.ErrActiveMatchNotFound) {
		log.Printf("match resume failed: player_id = %d err = %v", session.PlayerID, err)
	}

	for {
		readCtx, cancel := context.WithTimeout(context.Background(), websocketReadTimeout)
		msgType, msg, err := conn.Read(readCtx)
		cancel()
		if err != nil {
			return
		}
		if msgType != websocket.MessageText {
			continue
		}
		var message websocketMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			continue
		}
		switch message.Type {
		case messageTypeHeartbeat:
			// WebSocket 心跳同时刷新 Redis presence TTL 和本地连接时间；任一失败都
			// 结束连接，避免本地与跨实例在线视图长期不一致。
			if err := h.presenceService.Refresh(context.Background(), session.PlayerID, h.serverName); err != nil {
				return
			}
			if !h.connections.Touch(connInfo.playerID, connInfo.id, time.Now()) {
				return
			}
		case messageTypeMatchStart:
			if h.matchService == nil {
				continue
			}
			res, err := h.matchService.Start(context.Background(), session.PlayerID, message.Weapon)
			if err != nil {
				_ = wsjson.Write(context.Background(), conn, matchErrorMessage{
					Type:  serverEventMatchError,
					Error: err.Error(),
				})
				continue
			}
			_ = wsjson.Write(context.Background(), conn, matchResultMessage{
				Type:           serverEventMatchResult,
				Status:         string(res.Status),
				RoomName:       res.RoomName,
				Token:          res.Token,
				BattleNodeName: res.BattleNodeName,
				BattleUDPAddr:  res.BattleUDPAddr,
			})
			// 当前连接已直接收到结果；另一名匹配玩家可能位于不同 logic-server，
			// 因此再按其 presence 将结果发到对应的实时频道。
			h.pushMatchResultToPlayers(context.Background(), session.PlayerID, res)
		case messageTypeMatchCancel:
			if h.matchService == nil {
				continue
			}
			if err := h.matchService.Cancel(context.Background(), session.PlayerID); err != nil {
				_ = wsjson.Write(context.Background(), conn, matchErrorMessage{
					Type:  serverEventMatchError,
					Error: err.Error(),
				})
				continue
			}
			_ = wsjson.Write(context.Background(), conn, matchCancelMessage{
				Type: serverEventMatchCanceled,
			})
		case messageTypeMatchResume:
			if h.matchService == nil {
				continue
			}
			err := h.sendResumedMatch(context.Background(), conn, session.PlayerID)
			if err != nil {
				_ = wsjson.Write(context.Background(), conn, matchErrorMessage{
					Type:  serverEventMatchError,
					Error: err.Error(),
				})
				continue
			}
		}
	}
}

// sendResumedMatch 向玩家的 WebSocket 连接发送其活跃战斗分配。
func (h *Handler) sendResumedMatch(ctx context.Context, conn *websocket.Conn, playerID int64) error {
	if h.matchService == nil {
		return nil
	}
	res, err := h.matchService.Resume(ctx, playerID)
	if err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, matchResultMessage{
		Type:           serverEventMatchResult,
		Status:         string(res.Status),
		RoomName:       res.RoomName,
		Token:          res.Token,
		BattleNodeName: res.BattleNodeName,
		BattleUDPAddr:  res.BattleUDPAddr,
	})
}

// pushMatchResultToPlayers 向房间内其他玩家发送匹配结果。
func (h *Handler) pushMatchResultToPlayers(ctx context.Context, currentPlayerID int64, result *rcenter.MatchResult) {
	if result == nil || result.Status != rcenter.MatchStatusMatched {
		return
	}
	msg := matchResultMessage{
		Type:           serverEventMatchResult,
		Status:         string(result.Status),
		RoomName:       result.RoomName,
		Token:          result.Token,
		BattleNodeName: result.BattleNodeName,
		BattleUDPAddr:  result.BattleUDPAddr,
	}
	for _, playerID := range result.PlayerIDs {
		if playerID == currentPlayerID {
			continue
		}
		// 优先走同实例内存连接；不在本机时才查询 presence 并借 Redis 实时频道转发。
		if h.connections.SendJSON(ctx, playerID, msg) {
			continue
		}
		if h.realtimeClient == nil {
			continue
		}

		playerPresence, err := h.presenceService.Get(ctx, playerID)
		if err != nil {
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
		_ = h.realtimeClient.PublishRealtimeToServer(ctx, playerPresence.ServerName, event)
	}
}
