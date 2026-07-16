package httpapi

import "net/http"

// Routes builds all HTTP routes for the game server API.
func (h *Handler) Routes() http.Handler {
	var mux = http.NewServeMux()
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /players", h.handleCreatePlayer)
	mux.HandleFunc("GET /players/{id}", h.handleGetPlayer)
	mux.HandleFunc("PATCH /players/{id}", h.handleUpdatePlayer)
	mux.HandleFunc("POST /rooms", h.handleCreateRoom)
	mux.HandleFunc("GET /rooms/{id}", h.handleGetRoom)
	mux.HandleFunc("GET /rooms", h.handleListRoom)
	mux.HandleFunc("POST /rooms/{id}/join", h.handleJoinRoom)
	mux.HandleFunc("POST /rooms/{id}/leave", h.handleLeaveRoom)
	mux.HandleFunc("POST /rooms/{id}/ready", h.handleReadyRoom)
	mux.HandleFunc("POST /rooms/{id}/unready", h.handleUnreadyRoom)
	mux.HandleFunc("POST /rooms/{id}/start", h.handleStartRoom)
	return mux
}
