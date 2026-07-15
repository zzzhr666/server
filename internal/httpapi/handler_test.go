package httpapi

import (
	"context"
	"encoding/json"
	playerpkg "learning/internal/player"
	roompkg "learning/internal/room"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

var testCtx = context.Background()

func TestHealth(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want %d; got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("want %s; got %s", "ok", rec.Body.String())
	}
}

func TestCreatePlayerHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodPost, "/players", strings.NewReader(`{"name":"alice"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %q, want %q", contentType, "application/json")
	}

	var resp playerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != 1 {
		t.Fatalf("player id = %d, want 1", resp.ID)
	}

	if resp.Name != "alice" {
		t.Fatalf("player name = %q, want %q", resp.Name, "alice")
	}
}

func TestCreatePlayerHTTPInvalidName(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodPost, "/players", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %q, want %q", contentType, "application/json")
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("error response is empty")
	}
}

func TestCreatePlayerHTTPInvalidJSON(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodPost, "/players", strings.NewReader(`{"name":`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %q, want %q", contentType, "application/json")
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("error response is empty")
	}
}

func TestGetPlayerHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	player, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodGet, "/players/1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %q, want %q", contentType, "application/json")
	}

	var resp playerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != player.ID {
		t.Fatalf("player id = %d, want %d", resp.ID, player.ID)
	}

	if resp.Name != player.Name {
		t.Fatalf("player name = %q, want %q", resp.Name, player.Name)
	}
}

func TestGetPlayerHTTPInvalidID(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodGet, "/players/not-a-number", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("error response is empty")
	}
}

func TestGetPlayerHTTPNotFound(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodGet, "/players/999", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("error response is empty")
	}
}

func TestCreateRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"owner_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp roomResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != 1 {
		t.Fatalf("room id = %d, want 1", resp.ID)
	}

	if resp.OwnerID != owner.ID {
		t.Fatalf("room owner id = %d, want %d", resp.OwnerID, owner.ID)
	}

	if !reflect.DeepEqual(resp.Players, []int64{owner.ID}) {
		t.Fatalf("players = %#v, want %#v", resp.Players, []int64{owner.ID})
	}
}

func TestCreateRoomHTTPOwnerNotFound(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"owner_id":999}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("error response is empty")
	}
}

func TestGetRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodGet, "/rooms/1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp roomResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != room.ID {
		t.Fatalf("room id = %d, want %d", resp.ID, room.ID)
	}

	if resp.OwnerID != owner.ID {
		t.Fatalf("room owner id = %d, want %d", resp.OwnerID, owner.ID)
	}

	if !reflect.DeepEqual(resp.Players, []int64{owner.ID}) {
		t.Fatalf("players = %#v, want %#v", resp.Players, []int64{owner.ID})
	}
}

func TestGetRoomHTTPNotFound(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodGet, "/rooms/999", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestListRoomsHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	alice, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	bob, err := players.Create(testCtx, "bob")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room1, err := rooms.Create(testCtx, alice.ID)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	room2, err := rooms.Create(testCtx, bob.ID)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodGet, "/rooms", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp listRoomsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Rooms) != 2 {
		t.Fatalf("room count = %d, want 2", len(resp.Rooms))
	}

	roomsByID := make(map[int64]roomResponse, len(resp.Rooms))
	for _, room := range resp.Rooms {
		roomsByID[room.ID] = room
	}

	if _, ok := roomsByID[room1.ID]; !ok {
		t.Fatalf("room id %d not found", room1.ID)
	}

	if _, ok := roomsByID[room2.ID]; !ok {
		t.Fatalf("room id %d not found", room2.ID)
	}
}

func TestJoinRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, "bob")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/join", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom returned error: %v", err)
	}

	if _, ok := got.Players[player.ID]; !ok {
		t.Fatalf("player id %d is not in room players", player.ID)
	}
}

func TestJoinRoomHTTPAlreadyInRoom(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if _, err := rooms.Create(testCtx, owner.ID); err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/join", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestLeaveRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, "bob")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/leave", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom returned error: %v", err)
	}

	if _, ok := got.Players[player.ID]; ok {
		t.Fatalf("player id %d is still in room players", player.ID)
	}
}

func TestLeaveRoomHTTPNotInRoom(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, "alice")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, "bob")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if _, err := rooms.Create(testCtx, owner.ID); err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/leave", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	_ = player
}

type fakePlayerService struct {
	nextPlayerID int64
	players      map[int64]*playerpkg.Player
}

type fakeRoomService struct {
	nextRoomID int64
	players    *fakePlayerService
	rooms      map[int64]*roompkg.Room
}

func newFakeServices() (*fakePlayerService, *fakeRoomService) {
	players := &fakePlayerService{
		nextPlayerID: 1,
		players:      make(map[int64]*playerpkg.Player),
	}
	rooms := &fakeRoomService{
		nextRoomID: 1,
		players:    players,
		rooms:      make(map[int64]*roompkg.Room),
	}
	return players, rooms
}

func (s *fakePlayerService) Create(ctx context.Context, name string) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, playerpkg.ErrInvalidName
	}
	id := s.nextPlayerID
	s.nextPlayerID++
	player := &playerpkg.Player{ID: id, Name: name}
	s.players[id] = player
	return &playerpkg.Player{ID: player.ID, Name: player.Name}, nil
}

func (s *fakePlayerService) Get(ctx context.Context, id int64) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	player, ok := s.players[id]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	return &playerpkg.Player{ID: player.ID, Name: player.Name}, nil
}

func (s *fakeRoomService) Create(ctx context.Context, ownerID int64) (*roompkg.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, ok := s.players.players[ownerID]; !ok {
		return nil, roompkg.ErrPlayerNotFound
	}
	id := s.nextRoomID
	s.nextRoomID++
	room := &roompkg.Room{
		ID:      id,
		OwnerID: ownerID,
		Players: map[int64]struct{}{
			ownerID: {},
		},
	}
	s.rooms[id] = room
	return cloneRoom(room), nil
}

func (s *fakeRoomService) Get(ctx context.Context, roomID int64) (*roompkg.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	room, ok := s.rooms[roomID]
	if !ok {
		return nil, roompkg.ErrNotFound
	}
	return cloneRoom(room), nil
}

func (s *fakeRoomService) List(ctx context.Context) ([]*roompkg.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rooms := make([]*roompkg.Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, cloneRoom(room))
	}
	return rooms, nil
}

func (s *fakeRoomService) Join(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := s.players.players[playerID]; !ok {
		return roompkg.ErrPlayerNotFound
	}
	room, ok := s.rooms[roomID]
	if !ok {
		return roompkg.ErrNotFound
	}
	if _, ok := room.Players[playerID]; ok {
		return roompkg.ErrPlayerAlreadyIn
	}
	room.Players[playerID] = struct{}{}
	return nil
}

func (s *fakeRoomService) Leave(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := s.players.players[playerID]; !ok {
		return roompkg.ErrPlayerNotFound
	}
	room, ok := s.rooms[roomID]
	if !ok {
		return roompkg.ErrNotFound
	}
	if _, ok := room.Players[playerID]; !ok {
		return roompkg.ErrPlayerNotIn
	}
	delete(room.Players, playerID)
	return nil
}

func cloneRoom(room *roompkg.Room) *roompkg.Room {
	return &roompkg.Room{
		ID:      room.ID,
		OwnerID: room.OwnerID,
		Players: maps.Clone(room.Players),
	}
}
