package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/presence"
	"time"
)

func (h *Handler) currentPlayerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	token, ok := bearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return 0, false
	}
	session, err := h.authService.GetSession(r.Context(), token)
	if errors.Is(err, auth.ErrSessionNotFound) {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return 0, false
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return 0, false
	}

	return session.PlayerID, true
}

func (h *Handler) handleSendRequest(w http.ResponseWriter, r *http.Request) {
	fromPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	var req sendFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if err := h.friendService.SendRequest(r.Context(), fromPlayerID, req.ToPlayerID); err != nil {
		writeFriendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListIncomingRequests(w http.ResponseWriter, r *http.Request) {
	currentPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	requests, err := h.friendService.ListIncomingRequests(r.Context(), currentPlayerID)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	respRequests := toFriendRequestResponses(requests)
	writeJSON(w, http.StatusOK, friendRequestsResponse{Requests: respRequests})
}

func (h *Handler) handleListOutgoingRequests(w http.ResponseWriter, r *http.Request) {
	currentPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	requests, err := h.friendService.ListOutgoingRequests(r.Context(), currentPlayerID)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	respRequests := toFriendRequestResponses(requests)
	writeJSON(w, http.StatusOK, friendRequestsResponse{Requests: respRequests})
}

func (h *Handler) handleAcceptRequest(w http.ResponseWriter, r *http.Request) {
	toPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	var req handleFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	if err := h.friendService.AcceptRequest(r.Context(), req.FromPlayerID, toPlayerID); err != nil {
		writeFriendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRejectRequest(w http.ResponseWriter, r *http.Request) {
	toPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	var req handleFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	if err := h.friendService.RejectRequest(r.Context(), req.FromPlayerID, toPlayerID); err != nil {
		writeFriendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListFriends(w http.ResponseWriter, r *http.Request) {
	currentPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	ids, err := h.friendService.ListFriendIDs(r.Context(), currentPlayerID)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	var friendSummaries []friendSummaryResponse
	for _, id := range ids {
		info := friendSummaryResponse{PlayerID: id}
		player, err := h.playerService.Get(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			return
		}
		info.Avatar = player.Avatar
		info.Nickname = player.Nickname
		info.Online = false
		info.Status = "offline"

		pres, err := h.presenceService.Get(r.Context(), id)
		if err == nil {
			info.Online = pres.Status == presence.StatusOnline
			info.Status = pres.Status
			info.UpdatedAt = pres.UpdatedAt.Format(time.RFC3339Nano)
		} else if !errors.Is(err, presence.ErrNotFound) {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			return
		}
		friendSummaries = append(friendSummaries, info)

	}

	writeJSON(w, http.StatusOK, friendSummariesResponse{Friends: friendSummaries})
}

func (h *Handler) handleDeleteFriend(w http.ResponseWriter, r *http.Request) {
	currentPlayerID, ok := h.currentPlayerID(w, r)
	if !ok {
		return
	}
	var req deleteFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if err := h.friendService.DeleteFriend(r.Context(), currentPlayerID, req.FriendPlayerID); err != nil {
		writeFriendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeFriendError(w http.ResponseWriter, err error) {
	if errors.Is(err, friend.ErrInvalidPlayerID) || errors.Is(err, friend.ErrInvalidRequest) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, friend.ErrRequestNotFound) || errors.Is(err, friend.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, friend.ErrRequestExists) || errors.Is(err, friend.ErrAlreadyExists) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

func toFriendRequestResponses(requests []*friend.Request) []friendRequestResponse {
	ret := make([]friendRequestResponse, 0, len(requests))
	for _, req := range requests {
		ret = append(ret, friendRequestResponse{
			FromPlayerID: req.FromPlayerID,
			ToPlayerID:   req.ToPlayerID,
			CreatedAt:    req.CreatedAt.Format(time.RFC3339Nano),
		})
	}
	return ret
}
