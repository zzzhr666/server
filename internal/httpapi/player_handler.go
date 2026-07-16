package httpapi

import (
	"encoding/json"
	"errors"
	playerpkg "learning/internal/player"
	"net/http"
)

// handleCreatePlayer creates a player from a JSON request body.
func (h *Handler) handleCreatePlayer(w http.ResponseWriter, r *http.Request) {
	var req createPlayerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	p, err := h.playersService.Create(r.Context(), playerpkg.CreateInput{
		Name:   req.Name,
		Avatar: req.Avatar,
		Email:  req.Email,
		Phone:  req.Phone,
	})
	if errors.Is(err, playerpkg.ErrInvalidName) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: playerpkg.ErrInvalidName.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	writeJSON(w, http.StatusCreated, toPlayerResponse(p))
}

// handleGetPlayer returns a player by path ID.
func (h *Handler) handleGetPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	p, err := h.playersService.Get(r.Context(), id)
	if errors.Is(err, playerpkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: playerpkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	writeJSON(w, http.StatusOK, toPlayerResponse(p))
}

// handleUpdatePlayer updates player profile fields from a JSON request body.
func (h *Handler) handleUpdatePlayer(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	req := updatePlayerRequest{}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	var profile = playerpkg.UpdateProfileInput{
		Name:   req.Name,
		Avatar: req.Avatar,
		Email:  req.Email,
		Phone:  req.Phone,
	}
	p, err := h.playersService.UpdateProfile(r.Context(), id, profile)
	if errors.Is(err, playerpkg.ErrInvalidName) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: playerpkg.ErrInvalidName.Error()})
		return
	} else if errors.Is(err, playerpkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: playerpkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	writeJSON(w, http.StatusOK, toPlayerResponse(p))
}
