package httpapi

import (
	"encoding/json"
	"net/http"
	playerpkg "server/internal/logic/player"
)

// writeJSON 使用给定状态码写入 JSON 响应。
func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}

// toPlayerResponse 将玩家领域模型转换为 HTTP 响应体。
func toPlayerResponse(player *playerpkg.Player) (r playerResponse) {
	r.ID = player.ID
	r.Nickname = player.Nickname
	r.Avatar = player.Avatar
	r.Email = player.Email
	r.Phone = player.Phone
	r.Coins = player.Coins
	return
}
