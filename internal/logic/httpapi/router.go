package httpapi

import "net/http"

// Routes 构建健康检查以及注册、登录所需的公共 HTTP 路由。
func (h *Handler) Routes() http.Handler {
	var mux = http.NewServeMux()
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /auth/register", h.handleRegisterAuth)
	mux.HandleFunc("POST /auth/login", h.handleLoginAuth)
	return mux
}
