package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
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

	if err := h.presenceService.MarkOnline(r.Context(), session.PlayerID, h.serverName); err != nil {
		if closeErr := conn.Close(websocket.StatusInternalError, "presence failed"); closeErr != nil {
			log.Printf("close websocket failed: player_id = %d err = %v", session.PlayerID, closeErr)
		}
		return
	}
	connInfo := h.connections.Add(session.PlayerID, conn)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if !h.connections.Remove(session.PlayerID, connInfo.id) {
			return
		}
		if err := h.presenceService.MarkOffline(ctx, session.PlayerID, h.serverName); err != nil {
			log.Printf("mark offline failed: player_id = %d server_name = %s,err = %v", session.PlayerID, h.serverName, err)
		}
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
		if message.Type == websocketMessageTypeHeartbeat {
			if err := h.presenceService.Refresh(context.Background(), session.PlayerID, h.serverName); err != nil {
				return
			}
			if !h.connections.Touch(connInfo.playerID, connInfo.id, time.Now()) {
				return
			}
		}
	}
}
