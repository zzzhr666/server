package httpapi

import (
	"fmt"
	"net/http"
)

// handleHealth 返回简单的存活探针响应。
func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("ok server_name = %v", h.serverName)))
}
