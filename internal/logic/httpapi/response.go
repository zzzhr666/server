package httpapi

import (
	"encoding/json"
	"net/http"
	playerpkg "server/internal/logic/player"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}

// toPlayerResponse converts a player domain model into an HTTP response body.
func toPlayerResponse(player *playerpkg.Player) (r playerResponse) {
	r.ID = player.ID
	r.Nickname = player.Nickname
	r.Avatar = player.Avatar
	r.Email = player.Email
	r.Phone = player.Phone
	return
}
