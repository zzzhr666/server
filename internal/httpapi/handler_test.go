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

	req := httptest.NewRequest(http.MethodPost, "/players", strings.NewReader(`{"name":"alice","avatar":"avatar.png","email":"alice@example.com","phone":"13800000000"}`))
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

	if resp.Avatar != "avatar.png" {
		t.Fatalf("player avatar = %q, want %q", resp.Avatar, "avatar.png")
	}

	if resp.Email != "alice@example.com" {
		t.Fatalf("player email = %q, want %q", resp.Email, "alice@example.com")
	}

	if resp.Phone != "13800000000" {
		t.Fatalf("player phone = %q, want %q", resp.Phone, "13800000000")
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
	player, err := players.Create(testCtx, playerpkg.CreateInput{
		Name:   "alice",
		Avatar: "avatar.png",
		Email:  "alice@example.com",
		Phone:  "13800000000",
	})
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

	if resp.Avatar != player.Avatar {
		t.Fatalf("player avatar = %q, want %q", resp.Avatar, player.Avatar)
	}

	if resp.Email != player.Email {
		t.Fatalf("player email = %q, want %q", resp.Email, player.Email)
	}

	if resp.Phone != player.Phone {
		t.Fatalf("player phone = %q, want %q", resp.Phone, player.Phone)
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

func TestUpdatePlayerHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	player, err := players.Create(testCtx, playerpkg.CreateInput{
		Name:   "alice",
		Avatar: "avatar.png",
		Email:  "alice@example.com",
		Phone:  "13800000000",
	})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPatch, "/players/1", strings.NewReader(`{"name":"alice2","email":"alice2@example.com"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp playerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != player.ID {
		t.Fatalf("player id = %d, want %d", resp.ID, player.ID)
	}
	if resp.Name != "alice2" {
		t.Fatalf("player name = %q, want %q", resp.Name, "alice2")
	}
	if resp.Email != "alice2@example.com" {
		t.Fatalf("player email = %q, want %q", resp.Email, "alice2@example.com")
	}
	if resp.Avatar != "avatar.png" {
		t.Fatalf("player avatar = %q, want %q", resp.Avatar, "avatar.png")
	}
	if resp.Phone != "13800000000" {
		t.Fatalf("player phone = %q, want %q", resp.Phone, "13800000000")
	}
}

func TestUpdatePlayerHTTPClearsOptionalFields(t *testing.T) {
	players, rooms := newFakeServices()
	if _, err := players.Create(testCtx, playerpkg.CreateInput{
		Name:   "alice",
		Avatar: "avatar.png",
		Email:  "alice@example.com",
		Phone:  "13800000000",
	}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPatch, "/players/1", strings.NewReader(`{"avatar":"","phone":""}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp playerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Avatar != "" {
		t.Fatalf("player avatar = %q, want empty string", resp.Avatar)
	}
	if resp.Phone != "" {
		t.Fatalf("player phone = %q, want empty string", resp.Phone)
	}
	if resp.Name != "alice" {
		t.Fatalf("player name = %q, want %q", resp.Name, "alice")
	}
	if resp.Email != "alice@example.com" {
		t.Fatalf("player email = %q, want %q", resp.Email, "alice@example.com")
	}
}

func TestUpdatePlayerHTTPInvalidName(t *testing.T) {
	players, rooms := newFakeServices()
	if _, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPatch, "/players/1", strings.NewReader(`{"name":""}`))
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

func TestUpdatePlayerHTTPInvalidID(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodPatch, "/players/not-a-number", strings.NewReader(`{"name":"alice2"}`))
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

func TestUpdatePlayerHTTPNotFound(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()

	req := httptest.NewRequest(http.MethodPatch, "/players/999", strings.NewReader(`{"name":"alice2"}`))
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

func TestUpdatePlayerHTTPInvalidJSON(t *testing.T) {
	players, rooms := newFakeServices()
	if _, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPatch, "/players/1", strings.NewReader(`{"name":`))
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

func TestCreateRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
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

	if resp.Status != string(roompkg.StatusWaiting) {
		t.Fatalf("room status = %q, want %q", resp.Status, roompkg.StatusWaiting)
	}

	if resp.MaxPlayers != fakeRoomMaxPlayers {
		t.Fatalf("max players = %d, want %d", resp.MaxPlayers, fakeRoomMaxPlayers)
	}

	if !reflect.DeepEqual(resp.Players, []int64{owner.ID}) {
		t.Fatalf("players = %#v, want %#v", resp.Players, []int64{owner.ID})
	}

	if !reflect.DeepEqual(resp.ReadyPlayers, []int64{}) {
		t.Fatalf("ready players = %#v, want empty slice", resp.ReadyPlayers)
	}
}

func TestCreateRoomHTTPWithMaxPlayers(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"owner_id":1,"max_players":5}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp roomResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.OwnerID != owner.ID {
		t.Fatalf("room owner id = %d, want %d", resp.OwnerID, owner.ID)
	}

	if resp.MaxPlayers != 5 {
		t.Fatalf("max players = %d, want 5", resp.MaxPlayers)
	}
}

func TestCreateRoomHTTPInvalidMaxPlayers(t *testing.T) {
	players, rooms := newFakeServices()
	if _, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"owner_id":1,"max_players":11}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != roompkg.ErrInvalidMaxPlayers.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrInvalidMaxPlayers.Error())
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

func TestCreateRoomHTTPOwnerAlreadyInAnotherRoom(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	if _, err := rooms.Create(testCtx, owner.ID, 0); err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"owner_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != roompkg.ErrPlayerAlreadyInAnotherRoom.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrPlayerAlreadyInAnotherRoom.Error())
	}
}

func TestGetRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{
		Name:   "alice",
		Avatar: "alice.png",
	})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	member, err := players.Create(testCtx, playerpkg.CreateInput{
		Name:   "bob",
		Avatar: "bob.png",
	})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	if err := rooms.Ready(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("ReadyRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodGet, "/rooms/1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp roomDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != room.ID {
		t.Fatalf("room id = %d, want %d", resp.ID, room.ID)
	}

	if resp.OwnerID != owner.ID {
		t.Fatalf("room owner id = %d, want %d", resp.OwnerID, owner.ID)
	}

	if resp.Status != string(roompkg.StatusWaiting) {
		t.Fatalf("room status = %q, want %q", resp.Status, roompkg.StatusWaiting)
	}

	if resp.MaxPlayers != fakeRoomMaxPlayers {
		t.Fatalf("max players = %d, want %d", resp.MaxPlayers, fakeRoomMaxPlayers)
	}

	wantPlayers := []roomPlayerResponse{
		{ID: owner.ID, Name: "alice", Avatar: "alice.png", Ready: false, IsOwner: true},
		{ID: member.ID, Name: "bob", Avatar: "bob.png", Ready: true, IsOwner: false},
	}
	if !reflect.DeepEqual(resp.Players, wantPlayers) {
		t.Fatalf("players = %#v, want %#v", resp.Players, wantPlayers)
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
	alice, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	bob, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room1, err := rooms.Create(testCtx, alice.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	room2, err := rooms.Create(testCtx, bob.ID, 0)
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
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
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
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if _, err := rooms.Create(testCtx, owner.ID, 0); err != nil {
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

func TestJoinRoomHTTPAlreadyInAnotherRoom(t *testing.T) {
	players, rooms := newFakeServices()
	owner1, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	owner2, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	member, err := players.Create(testCtx, playerpkg.CreateInput{Name: "carl"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	room1, err := rooms.Create(testCtx, owner1.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if _, err := rooms.Create(testCtx, owner2.ID, 0); err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, member.ID, room1.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/2/join", strings.NewReader(`{"player_id":3}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != roompkg.ErrPlayerAlreadyInAnotherRoom.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrPlayerAlreadyInAnotherRoom.Error())
	}
}

func TestJoinRoomHTTPRoomFull(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if _, err := rooms.Create(testCtx, owner.ID, 1); err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/join", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != roompkg.ErrRoomFull.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrRoomFull.Error())
	}

	_ = player
}

func TestJoinRoomHTTPRoomNotWaiting(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	rooms.rooms[room.ID].Status = roompkg.StatusPlaying

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/join", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != roompkg.ErrRoomNotWaiting.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrRoomNotWaiting.Error())
	}

	_ = player
}

func TestLeaveRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
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

func TestLeaveRoomHTTPClearsReadyPlayer(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	rooms.rooms[room.ID].ReadyPlayers[player.ID] = struct{}{}

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
	if _, ok := got.ReadyPlayers[player.ID]; ok {
		t.Fatalf("player id %d is still in ready players", player.ID)
	}
}

func TestLeaveRoomHTTPOwnerTransfersToSmallestPlayerID(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player2, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player3, err := players.Create(testCtx, playerpkg.CreateInput{Name: "carl"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player3.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player2.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/leave", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom returned error: %v", err)
	}
	if got.OwnerID != player2.ID {
		t.Fatalf("owner id = %d, want %d", got.OwnerID, player2.ID)
	}
}

func TestLeaveRoomHTTPLastPlayerDeletesRoom(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/leave", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	if _, err := rooms.Get(testCtx, room.ID); err != roompkg.ErrNotFound {
		t.Fatalf("GetRoom error = %v, want %v", err, roompkg.ErrNotFound)
	}
}

func TestLeaveRoomHTTPNotInRoom(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if _, err := rooms.Create(testCtx, owner.ID, 0); err != nil {
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

func TestReadyRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/ready", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom returned error: %v", err)
	}
	if _, ok := got.ReadyPlayers[player.ID]; !ok {
		t.Fatalf("player id %d is not ready", player.ID)
	}
}

func TestUnreadyRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	if err := rooms.Ready(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/unready", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom returned error: %v", err)
	}
	if _, ok := got.ReadyPlayers[player.ID]; ok {
		t.Fatalf("player id %d is still ready", player.ID)
	}
}

func TestReadyRoomHTTPOwnerCannotReady(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if _, err := rooms.Create(testCtx, owner.ID, 0); err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/ready", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != roompkg.ErrOwnerCannotReadyOrUnready.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrOwnerCannotReadyOrUnready.Error())
	}
}

func TestReadyRoomHTTPRoomNotFound(t *testing.T) {
	players, rooms := newFakeServices()
	if _, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/999/ready", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != roompkg.ErrNotFound.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrNotFound.Error())
	}
}

func TestReadyRoomHTTPInvalidJSON(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/ready", strings.NewReader(`{"player_id":`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestStartRoomHTTP(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}
	if err := rooms.Ready(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/start", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom returned error: %v", err)
	}
	if got.Status != roompkg.StatusPlaying {
		t.Fatalf("status = %q, want %q", got.Status, roompkg.StatusPlaying)
	}
}

func TestStartRoomHTTPNonOwner(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/start", strings.NewReader(`{"player_id":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != roompkg.ErrOnlyOwnerCanStart.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrOnlyOwnerCanStart.Error())
	}
}

func TestStartRoomHTTPPlayersNotReady(t *testing.T) {
	players, rooms := newFakeServices()
	owner, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	player, err := players.Create(testCtx, playerpkg.CreateInput{Name: "bob"})
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	room, err := rooms.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if err := rooms.Join(testCtx, player.ID, room.ID); err != nil {
		t.Fatalf("JoinRoom returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/start", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != roompkg.ErrPlayersNotReady.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrPlayersNotReady.Error())
	}
}

func TestStartRoomHTTPRoomNotFound(t *testing.T) {
	players, rooms := newFakeServices()
	if _, err := players.Create(testCtx, playerpkg.CreateInput{Name: "alice"}); err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/999/start", strings.NewReader(`{"player_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != roompkg.ErrNotFound.Error() {
		t.Fatalf("error = %q, want %q", resp.Error, roompkg.ErrNotFound.Error())
	}
}

func TestStartRoomHTTPInvalidJSON(t *testing.T) {
	players, rooms := newFakeServices()
	handler := NewHandler(players, rooms).Routes()
	req := httptest.NewRequest(http.MethodPost, "/rooms/1/start", strings.NewReader(`{"player_id":`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
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

const fakeRoomMaxPlayers = 10

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

func (s *fakePlayerService) Create(ctx context.Context, input playerpkg.CreateInput) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Name == "" {
		return nil, playerpkg.ErrInvalidName
	}
	id := s.nextPlayerID
	s.nextPlayerID++
	player := &playerpkg.Player{
		ID:     id,
		Name:   input.Name,
		Avatar: input.Avatar,
		Email:  input.Email,
		Phone:  input.Phone,
	}
	s.players[id] = player
	return cloneFakePlayer(player), nil
}

func (s *fakePlayerService) Get(ctx context.Context, id int64) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	player, ok := s.players[id]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	return cloneFakePlayer(player), nil
}

func (s *fakePlayerService) UpdateProfile(ctx context.Context, id int64, input playerpkg.UpdateProfileInput) (*playerpkg.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	player, ok := s.players[id]
	if !ok {
		return nil, playerpkg.ErrNotFound
	}
	if input.Name != nil {
		if *input.Name == "" {
			return nil, playerpkg.ErrInvalidName
		}
		player.Name = *input.Name
	}
	if input.Avatar != nil {
		player.Avatar = *input.Avatar
	}
	if input.Email != nil {
		player.Email = *input.Email
	}
	if input.Phone != nil {
		player.Phone = *input.Phone
	}
	return cloneFakePlayer(player), nil
}

func cloneFakePlayer(player *playerpkg.Player) *playerpkg.Player {
	return &playerpkg.Player{
		ID:     player.ID,
		Name:   player.Name,
		Avatar: player.Avatar,
		Email:  player.Email,
		Phone:  player.Phone,
	}
}

func (s *fakeRoomService) Create(ctx context.Context, ownerID int64, maxPlayers int) (*roompkg.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, ok := s.players.players[ownerID]; !ok {
		return nil, roompkg.ErrPlayerNotFound
	}
	if _, ok := s.playerRoomID(ownerID); ok {
		return nil, roompkg.ErrPlayerAlreadyInAnotherRoom
	}
	if maxPlayers == 0 {
		maxPlayers = fakeRoomMaxPlayers
	}
	if maxPlayers < 1 || maxPlayers > fakeRoomMaxPlayers {
		return nil, roompkg.ErrInvalidMaxPlayers
	}
	id := s.nextRoomID
	s.nextRoomID++
	room := &roompkg.Room{
		ID:         id,
		OwnerID:    ownerID,
		Status:     roompkg.StatusWaiting,
		MaxPlayers: maxPlayers,
		Players: map[int64]struct{}{
			ownerID: {},
		},
		ReadyPlayers: make(map[int64]struct{}),
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
	if currentRoomID, ok := s.playerRoomID(playerID); ok {
		if currentRoomID == roomID {
			return roompkg.ErrPlayerAlreadyInThisRoom
		}
		return roompkg.ErrPlayerAlreadyInAnotherRoom
	}
	if room.Status != roompkg.StatusWaiting {
		return roompkg.ErrRoomNotWaiting
	}
	if len(room.Players) >= room.MaxPlayers {
		return roompkg.ErrRoomFull
	}
	if _, ok := room.Players[playerID]; ok {
		return roompkg.ErrPlayerAlreadyInThisRoom
	}
	room.Players[playerID] = struct{}{}
	return nil
}

func (s *fakeRoomService) playerRoomID(playerID int64) (int64, bool) {
	for roomID, room := range s.rooms {
		if _, ok := room.Players[playerID]; ok {
			return roomID, true
		}
	}
	return 0, false
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
	delete(room.ReadyPlayers, playerID)
	if len(room.Players) == 0 {
		delete(s.rooms, roomID)
		return nil
	}
	if room.OwnerID == playerID {
		room.OwnerID = minFakePlayerID(room.Players)
	}
	return nil
}

func (s *fakeRoomService) Ready(ctx context.Context, playerID, roomID int64) error {
	return s.setReady(ctx, playerID, roomID, true)
}

func (s *fakeRoomService) Unready(ctx context.Context, playerID, roomID int64) error {
	return s.setReady(ctx, playerID, roomID, false)
}

func (s *fakeRoomService) Start(ctx context.Context, playerID, roomID int64) error {
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
	if room.Status != roompkg.StatusWaiting {
		return roompkg.ErrRoomNotWaiting
	}
	if room.OwnerID != playerID {
		return roompkg.ErrOnlyOwnerCanStart
	}
	for memberID := range room.Players {
		if memberID == room.OwnerID {
			continue
		}
		if _, ok := room.ReadyPlayers[memberID]; !ok {
			return roompkg.ErrPlayersNotReady
		}
	}
	room.Status = roompkg.StatusPlaying
	return nil
}

func (s *fakeRoomService) setReady(ctx context.Context, playerID, roomID int64, ready bool) error {
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
	if room.Status != roompkg.StatusWaiting {
		return roompkg.ErrRoomNotWaiting
	}
	if _, ok := room.Players[playerID]; !ok {
		return roompkg.ErrPlayerNotIn
	}
	if playerID == room.OwnerID {
		return roompkg.ErrOwnerCannotReadyOrUnready
	}
	if ready {
		room.ReadyPlayers[playerID] = struct{}{}
	} else {
		delete(room.ReadyPlayers, playerID)
	}
	return nil
}

func minFakePlayerID(players map[int64]struct{}) int64 {
	var minID int64
	first := true
	for playerID := range players {
		if first || playerID < minID {
			minID = playerID
			first = false
		}
	}
	return minID
}

func cloneRoom(room *roompkg.Room) *roompkg.Room {
	return &roompkg.Room{
		ID:           room.ID,
		OwnerID:      room.OwnerID,
		Status:       room.Status,
		MaxPlayers:   room.MaxPlayers,
		Players:      maps.Clone(room.Players),
		ReadyPlayers: maps.Clone(room.ReadyPlayers),
	}
}
