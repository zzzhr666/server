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
	return mux
}
