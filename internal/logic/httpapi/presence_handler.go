package httpapi

import (
	"context"
	"encoding/json"
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
		if !h.connections.Remove(session.PlayerID, connInfo.id) {
			return
		}
		if err := h.presenceService.MarkOffline(ctx, session.PlayerID, h.serverName); err != nil {
			log.Printf("mark offline failed: player_id = %d server_name = %s,err = %v", session.PlayerID, h.serverName, err)
			return
		}
		h.publishFriendPresenceChanged(ctx, session.PlayerID, false, presence.StatusOffline)
	}()

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
			res, err := h.matchService.Start(context.Background(), session.PlayerID)
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
				BattleKCPAddr:  res.BattleKCPAddr,
			})
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
		}
	}
}

// pushMatchResultToPlayers sends a matched result to the other players in the room.
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
		BattleKCPAddr:  result.BattleKCPAddr,
	}
	for _, playerID := range result.PlayerIDs {
		if playerID == currentPlayerID {
			continue
		}
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
			BattleKCPAddr:  result.BattleKCPAddr,
			MatchPlayerIDs: result.PlayerIDs,
		}
		_ = h.realtimeClient.PublishRealtimeToServer(ctx, playerPresence.ServerName, event)
	}
}
