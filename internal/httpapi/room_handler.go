package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	roompkg "learning/internal/room"
	"net/http"
	"sort"
)

// handleCreateRoom creates a room and adds the owner as the first player.
func (h *Handler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	room, err := h.roomsService.Create(r.Context(), req.OwnerID, req.MaxPlayers)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrInvalidMaxPlayers) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: roompkg.ErrInvalidMaxPlayers.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerAlreadyInAnotherRoom) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerAlreadyInAnotherRoom.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	writeJSON(w, http.StatusCreated, toRoomResponse(room))
}

// handleGetRoom returns a room by path ID.
func (h *Handler) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	room, err := h.roomsService.Get(r.Context(), roomID)
	if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	resp, err := h.toRoomDetailResponse(r.Context(), room)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleListRoom returns all rooms.
func (h *Handler) handleListRoom(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.roomsService.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	resp := listRoomsResponse{
		Rooms: make([]roomResponse, 0, len(rooms)),
	}
	for _, room := range rooms {
		resp.Rooms = append(resp.Rooms, toRoomResponse(room))
	}
	sort.Slice(resp.Rooms, func(i, j int) bool {
		return resp.Rooms[i].ID < resp.Rooms[j].ID
	})
	writeJSON(w, http.StatusOK, resp)
}

// handleJoinRoom adds a player to a room.
func (h *Handler) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	req := roomPlayerRequest{}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	playerID := req.PlayerID
	err = h.roomsService.Join(r.Context(), playerID, roomID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerAlreadyInThisRoom) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerAlreadyInThisRoom.Error()})
		return
	} else if errors.Is(err, roompkg.ErrRoomNotWaiting) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrRoomNotWaiting.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerAlreadyInAnotherRoom) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerAlreadyInAnotherRoom.Error()})
		return
	} else if errors.Is(err, roompkg.ErrRoomFull) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrRoomFull.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleLeaveRoom removes a player from a room.
func (h *Handler) handleLeaveRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	req := roomPlayerRequest{}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	playerID := req.PlayerID
	err = h.roomsService.Leave(r.Context(), playerID, roomID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerNotIn) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerNotIn.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleReadyRoom marks a non-owner room player as ready.
func (h *Handler) handleReadyRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	req := roomPlayerRequest{}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	playerID := req.PlayerID
	err = h.roomsService.Ready(r.Context(), playerID, roomID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrRoomNotWaiting) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrRoomNotWaiting.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerNotIn) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerNotIn.Error()})
		return
	} else if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrOwnerCannotReadyOrUnready) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrOwnerCannotReadyOrUnready.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleUnreadyRoom removes a non-owner room player from the ready set.
func (h *Handler) handleUnreadyRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	req := roomPlayerRequest{}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	playerID := req.PlayerID
	err = h.roomsService.Unready(r.Context(), playerID, roomID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrRoomNotWaiting) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrRoomNotWaiting.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerNotIn) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerNotIn.Error()})
		return
	} else if errors.Is(err, roompkg.ErrOwnerCannotReadyOrUnready) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrOwnerCannotReadyOrUnready.Error()})
		return
	} else if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleStartRoom starts a waiting room after owner and ready checks pass.
func (h *Handler) handleStartRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	req := roomPlayerRequest{}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	playerID := req.PlayerID
	err = h.roomsService.Start(r.Context(), playerID, roomID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrRoomNotWaiting) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrRoomNotWaiting.Error()})
		return
	} else if errors.Is(err, roompkg.ErrOnlyOwnerCanStart) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrOnlyOwnerCanStart.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayersNotReady) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayersNotReady.Error()})
		return
	} else if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal logicserver error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// toRoomResponse converts a room domain model into an HTTP response body.
func toRoomResponse(room *roompkg.Room) (r roomResponse) {
	r.ID = room.ID
	r.OwnerID = room.OwnerID
	r.Players = make([]int64, 0, len(room.Players))
	r.MaxPlayers = room.MaxPlayers
	r.ReadyPlayers = make([]int64, 0, len(room.ReadyPlayers))
	r.Status = string(room.Status)
	for k := range room.Players {
		r.Players = append(r.Players, k)
	}

	for k := range room.ReadyPlayers {
		r.ReadyPlayers = append(r.ReadyPlayers, k)
	}
	sort.Slice(r.Players, func(i, j int) bool { return r.Players[i] < r.Players[j] })
	sort.Slice(r.ReadyPlayers, func(i, j int) bool { return r.ReadyPlayers[i] < r.ReadyPlayers[j] })
	return r
}

// toRoomDetailResponse converts a room into a detailed response with player profiles.
func (h *Handler) toRoomDetailResponse(ctx context.Context, room *roompkg.Room) (roomDetailResponse, error) {
	resp := roomDetailResponse{
		ID:         room.ID,
		OwnerID:    room.OwnerID,
		Status:     string(room.Status),
		MaxPlayers: room.MaxPlayers,
		Players:    make([]roomPlayerResponse, 0, len(room.Players)),
	}
	playerIDs := make([]int64, 0, len(room.Players))
	for player := range room.Players {
		playerIDs = append(playerIDs, player)
	}
	sort.Slice(playerIDs, func(i, j int) bool { return playerIDs[i] < playerIDs[j] })
	for _, playerID := range playerIDs {
		p, err := h.playersService.Get(ctx, playerID)
		if err != nil {
			return roomDetailResponse{}, err
		}
		_, ready := room.ReadyPlayers[playerID]
		resp.Players = append(resp.Players, roomPlayerResponse{
			ID:      p.ID,
			Name:    p.Name,
			Avatar:  p.Avatar,
			Ready:   ready,
			IsOwner: p.ID == room.OwnerID,
		})
	}
	return resp, nil
}
