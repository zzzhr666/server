package httpapi

import (
	"encoding/json"
	playerpkg "learning/internal/player"
	"net/http"
	"strconv"
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
	r.Name = player.Name
	r.Avatar = player.Avatar
	r.Email = player.Email
	r.Phone = player.Phone
	return
}

// parseIDParam parses an int64 path value from a request.
func parseIDParam(r *http.Request, name string) (id int64, err error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
