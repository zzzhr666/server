package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"server/internal/logic/auth"
	"server/internal/logic/player"
)

func (h *Handler) handleRegisterAuth(w http.ResponseWriter, r *http.Request) {
	var req authRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	authRes, err := h.authService.Register(r.Context(), auth.RegisterInput{
		Username:      req.Username,
		PlainPassword: req.Password,
		Nickname:      req.Nickname,
		Avatar:        req.Avatar,
		Email:         req.Email,
		Phone:         req.Phone,
	})
	if errors.Is(err, auth.ErrInvalidUsername) || errors.Is(err, auth.ErrInvalidPassword) ||
		errors.Is(err, player.ErrInvalidNickname) || errors.Is(err, player.ErrInvalidAvatar) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	} else if errors.Is(err, auth.ErrAccountExists) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "account already exists"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, authSessionResponse{
		Token:  authRes.Session.Token,
		Player: toPlayerResponse(authRes.Player),
	})
}

func (h *Handler) handleLoginAuth(w http.ResponseWriter, r *http.Request) {
	var req authLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	authRes, err := h.authService.Login(r.Context(), auth.LoginInput{
		Username:      req.Username,
		PlainPassword: req.Password,
	})
	if errors.Is(err, auth.ErrInvalidUsername) || errors.Is(err, auth.ErrInvalidPassword) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	} else if errors.Is(err, auth.ErrInvalidCredentials) {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, authSessionResponse{
		Token:  authRes.Session.Token,
		Player: toPlayerResponse(authRes.Player),
	})

}
