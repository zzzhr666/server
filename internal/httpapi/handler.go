package httpapi

import (
	"encoding/json"
	"errors"
	playerpkg "learning/internal/player"
	roompkg "learning/internal/room"
	"net/http"
	"sort"
	"strconv"
)

type Handler struct {
	players playerpkg.Service
	rooms   roompkg.Service
}

func NewHandler(players playerpkg.Service, rooms roompkg.Service) *Handler {
	return &Handler{players: players, rooms: rooms}
}

type createPlayerRequest struct {
	Name string `json:"name"`
}

type roomPlayerRequest struct {
	PlayerID int64 `json:"player_id"`
}

type playerResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type createRoomRequest struct {
	OwnerID int64 `json:"owner_id"`
}

type roomResponse struct {
	ID      int64   `json:"id"`
	OwnerID int64   `json:"owner_id"`
	Players []int64 `json:"players"`
}

type listRoomsResponse struct {
	Rooms []roomResponse `json:"rooms"`
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /players", h.handleCreatePlayer)
	mux.HandleFunc("GET /players/{id}", h.handleGetPlayer)
	mux.HandleFunc("POST /rooms", h.handleCreateRoom)
	mux.HandleFunc("GET /rooms/{id}", h.handleGetRoom)
	mux.HandleFunc("GET /rooms", h.handleListRoom)
	mux.HandleFunc("POST /rooms/{id}/join", h.handleJoinRoom)
	mux.HandleFunc("POST /rooms/{id}/leave", h.handleLeaveRoom)
	return mux
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) handleCreatePlayer(w http.ResponseWriter, r *http.Request) {
	var req createPlayerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	p, err := h.players.Create(r.Context(), req.Name)
	if errors.Is(err, playerpkg.ErrInvalidName) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: playerpkg.ErrInvalidName.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, playerResponse{ID: p.ID, Name: p.Name})
}

func (h *Handler) handleGetPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	p, err := h.players.Get(r.Context(), id)
	if errors.Is(err, playerpkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: playerpkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, playerResponse{ID: p.ID, Name: p.Name})

}

func (h *Handler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	room, err := h.rooms.Create(r.Context(), req.OwnerID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, toRoomResponse(room))
}

func (h *Handler) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseIDParam(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ID"})
		return
	}
	room, err := h.rooms.Get(r.Context(), roomID)
	if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, toRoomResponse(room))
}

func (h *Handler) handleListRoom(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.rooms.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
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
	err = h.rooms.Join(r.Context(), playerID, roomID)
	if errors.Is(err, roompkg.ErrPlayerNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrPlayerNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: roompkg.ErrNotFound.Error()})
		return
	} else if errors.Is(err, roompkg.ErrPlayerAlreadyIn) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: roompkg.ErrPlayerAlreadyIn.Error()})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleLeaveRoom(w http.ResponseWriter, r *http.Request) {
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
	err = h.rooms.Leave(r.Context(), playerID, roomID)
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
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}

func toRoomResponse(room *roompkg.Room) (r roomResponse) {
	r.ID = room.ID
	r.OwnerID = room.OwnerID
	r.Players = make([]int64, 0, len(room.Players))
	for k := range room.Players {
		r.Players = append(r.Players, k)
	}

	return r
}

func parseIDParam(r *http.Request, name string) (id int64, err error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
