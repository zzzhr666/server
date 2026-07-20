package httpapi

import (
	"server/internal/logic/auth"
	"server/internal/logic/presence"
)

// Handler owns the HTTP and WebSocket routes for a logic-server instance.
type Handler struct {
	authService     auth.Service
	serverName      string
	presenceService presence.Service
	connections     *connManager
}

// HandlerConfig wires logic services into the HTTP adapter.
type HandlerConfig struct {
	AuthService     auth.Service
	ServerName      string
	PresenceService presence.Service
}

// NewHandler creates an HTTP handler with logic-server services.
func NewHandler(handlerConfig HandlerConfig) *Handler {
	return &Handler{
		authService:     handlerConfig.AuthService,
		serverName:      handlerConfig.ServerName,
		presenceService: handlerConfig.PresenceService,
		connections:     newConnManager(),
	}
}
