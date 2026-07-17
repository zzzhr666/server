package httpapi

import (
	"server/internal/logic/auth"
)

type Handler struct {
	authService auth.Service
}

type HandlerConfig struct {
	AuthService auth.Service
}

// NewHandler creates an HTTP handler with logic-server services.
func NewHandler(handlerConfig HandlerConfig) *Handler {
	return &Handler{
		authService: handlerConfig.AuthService,
	}
}
