package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"server/internal/logic/growth"
)

func toGrowthResponse(g *growth.Growth) growthResponse {
	return growthResponse{
		PlayerID:         g.PlayerID,
		AttackLevel:      g.AttackLevel,
		AttackSpeedLevel: g.AttackSpeedLevel,
		HealthLevel:      g.HealthLevel,
		MoveSpeedLevel:   g.MoveSpeedLevel,
	}
}

func parseUpgradeType(value string) (growth.UpgradeType, error) {
	switch value {
	case "attack":
		return growth.UpgradeAttack, nil
	case "attack_speed":
		return growth.UpgradeAttackSpeed, nil
	case "health":
		return growth.UpgradeHealth, nil
	case "move_speed":
		return growth.UpgradeMoveSpeed, nil
	default:
		return 0, growth.ErrInvalidUpgradeType
	}
}

func writeGrowthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, growth.ErrInvalidPlayerID),
		errors.Is(err, growth.ErrInvalidUpgradeType),
		errors.Is(err, growth.ErrInvalidGrowthLevel):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, growth.ErrGrowthNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, growth.ErrInsufficientCoins),
		errors.Is(err, growth.ErrMaxLevelReached):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func (h *Handler) handleGetGrowth(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	g, err := h.growthService.Get(r.Context(), playerID)
	if err != nil {
		writeGrowthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toGrowthResponse(g))
}

func (h *Handler) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	var req upgradeGrowthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	upgradeType, err := parseUpgradeType(req.Type)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := h.growthService.Upgrade(r.Context(), playerID, upgradeType)
	if err != nil {
		writeGrowthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, upgradeGrowthResponse{
		Growth:         toGrowthResponse(result.Growth),
		RemainingCoins: result.RemainingCoins,
		Cost:           result.Cost,
	})
}
