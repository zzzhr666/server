package httpapi

import "server/internal/logic/auth"

// Handler 持有一个 logic-server 实例的 HTTP 路由。
type Handler struct {
	authService auth.Service
	serverName  string
}

// HandlerConfig 将逻辑服务注入 HTTP 适配器。
type HandlerConfig struct {
	AuthService auth.Service
	ServerName  string
}

// NewHandler 使用 logic-server 服务创建 HTTP 处理器。
func NewHandler(handlerConfig HandlerConfig) *Handler {
	return &Handler{
		authService: handlerConfig.AuthService,
		serverName:  handlerConfig.ServerName,
	}
}
