package httpapi

import (
	"fmt"
	"net/http"
)

// handleHealth returns a simple liveness response.
func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("ok server_name = %v", h.serverName)))
}
