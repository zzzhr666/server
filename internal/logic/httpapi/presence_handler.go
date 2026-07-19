package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

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

	defer func(conn *websocket.Conn) {
		_ = conn.CloseNow()
	}(conn)

	if err := h.presenceService.MarkOnline(r.Context(), session.PlayerID, h.serverName); err != nil {
		if closeErr := conn.Close(websocket.StatusInternalError, "presence failed"); closeErr != nil {
			log.Printf("close websocket failed: player_id = %d err = %v", session.PlayerID, closeErr)
		}
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := h.presenceService.MarkOffline(ctx, session.PlayerID, h.serverName); err != nil {
			log.Printf("mark offline failed: player_id = %d server_name = %s,err = %v", session.PlayerID, h.serverName, err)
		}
	}()

	for {
		_, _, err := conn.Read(context.Background())
		if err != nil {
			return
		}
	}
}
