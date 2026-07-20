package httpapi

import "net/http"

// Routes builds all HTTP routes for the game server API.
func (h *Handler) Routes() http.Handler {
	var mux = http.NewServeMux()
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /auth/register", h.handleRegisterAuth)
	mux.HandleFunc("POST /auth/login", h.handleLoginAuth)
	mux.HandleFunc("POST /auth/logout", h.handleLogoutAuth)
	mux.HandleFunc("GET /auth/me", h.handleMeAuth)

	mux.HandleFunc("GET /ws", h.handleWebSocket)

	mux.HandleFunc("POST /friends/requests", h.handleSendRequest)
	mux.HandleFunc("GET /friends/requests/incoming", h.handleListIncomingRequests)
	mux.HandleFunc("GET /friends/requests/outgoing", h.handleListOutgoingRequests)
	mux.HandleFunc("POST /friends/requests/accept", h.handleAcceptRequest)
	mux.HandleFunc("POST /friends/requests/reject", h.handleRejectRequest)
	mux.HandleFunc("GET /friends", h.handleListFriends)
	mux.HandleFunc("DELETE /friends", h.handleDeleteFriend)
	return mux
}
