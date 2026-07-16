package room

import (
	"context"
	"errors"
	"learning/internal/player"
	"maps"
	"sync"
	"testing"
)

var testCtx = context.Background()

func TestReadyAddsReadyPlayer(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}

	if err := service.Ready(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if _, ok := got.ReadyPlayers[member.ID]; !ok {
		t.Fatalf("player id %d is not ready", member.ID)
	}
}

func TestUnreadyRemovesReadyPlayer(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}
	if err := service.Ready(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}

	if err := service.Unready(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Unready returned error: %v", err)
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if _, ok := got.ReadyPlayers[member.ID]; ok {
		t.Fatalf("player id %d is still ready", member.ID)
	}
	if _, ok := got.Players[member.ID]; !ok {
		t.Fatalf("player id %d was removed from room", member.ID)
	}
}

func TestReadyRejectsOwner(t *testing.T) {
	players, _, service := newTestRoomService()
	owner := players.add("alice")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	err = service.Ready(testCtx, owner.ID, room.ID)
	if !errors.Is(err, ErrOwnerCannotReadyOrUnready) {
		t.Fatalf("Ready error = %v, want %v", err, ErrOwnerCannotReadyOrUnready)
	}
}

func TestReadyRejectsPlayerNotInRoom(t *testing.T) {
	players, _, service := newTestRoomService()
	owner := players.add("alice")
	outsider := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	err = service.Ready(testCtx, outsider.ID, room.ID)
	if !errors.Is(err, ErrPlayerNotIn) {
		t.Fatalf("Ready error = %v, want %v", err, ErrPlayerNotIn)
	}
}

func TestReadyRejectsNonWaitingRoom(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}
	rooms.rooms[room.ID].Status = StatusPlaying

	err = service.Ready(testCtx, member.ID, room.ID)
	if !errors.Is(err, ErrRoomNotWaiting) {
		t.Fatalf("Ready error = %v, want %v", err, ErrRoomNotWaiting)
	}
}

func TestStartSetsRoomPlaying(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}
	if err := service.Ready(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}

	if err := service.Start(testCtx, owner.ID, room.ID); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusPlaying {
		t.Fatalf("status = %q, want %q", got.Status, StatusPlaying)
	}
}

func TestStartRejectsNonOwner(t *testing.T) {
	players, _, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}

	err = service.Start(testCtx, member.ID, room.ID)
	if !errors.Is(err, ErrOnlyOwnerCanStart) {
		t.Fatalf("Start error = %v, want %v", err, ErrOnlyOwnerCanStart)
	}
}

func TestStartRejectsUnreadyPlayers(t *testing.T) {
	players, _, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}

	err = service.Start(testCtx, owner.ID, room.ID)
	if !errors.Is(err, ErrPlayersNotReady) {
		t.Fatalf("Start error = %v, want %v", err, ErrPlayersNotReady)
	}
}

func TestStartAllowsOwnerAlone(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := service.Start(testCtx, owner.ID, room.ID); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	got, err := rooms.Get(testCtx, room.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusPlaying {
		t.Fatalf("status = %q, want %q", got.Status, StatusPlaying)
	}
}

func TestCreateSetsOwnerRoomIndex(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")

	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	roomID, ok, err := rooms.FindRoomByPlayer(testCtx, owner.ID)
	if err != nil {
		t.Fatalf("FindRoomByPlayer returned error: %v", err)
	}
	if !ok {
		t.Fatalf("owner room index was not set")
	}
	if roomID != room.ID {
		t.Fatalf("indexed room id = %d, want %d", roomID, room.ID)
	}
}

func TestCreateRejectsOwnerAlreadyInRoom(t *testing.T) {
	players, _, service := newTestRoomService()
	owner := players.add("alice")
	if _, err := service.Create(testCtx, owner.ID, 0); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err := service.Create(testCtx, owner.ID, 0)
	if !errors.Is(err, ErrPlayerAlreadyInAnotherRoom) {
		t.Fatalf("Create error = %v, want %v", err, ErrPlayerAlreadyInAnotherRoom)
	}
}

func TestJoinSetsPlayerRoomIndex(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}

	roomID, ok, err := rooms.FindRoomByPlayer(testCtx, member.ID)
	if err != nil {
		t.Fatalf("FindRoomByPlayer returned error: %v", err)
	}
	if !ok {
		t.Fatalf("member room index was not set")
	}
	if roomID != room.ID {
		t.Fatalf("indexed room id = %d, want %d", roomID, room.ID)
	}
}

func TestJoinRejectsPlayerAlreadyInAnotherRoom(t *testing.T) {
	players, _, service := newTestRoomService()
	owner1 := players.add("alice")
	owner2 := players.add("bob")
	member := players.add("carl")
	room1, err := service.Create(testCtx, owner1.ID, 0)
	if err != nil {
		t.Fatalf("Create room1 returned error: %v", err)
	}
	room2, err := service.Create(testCtx, owner2.ID, 0)
	if err != nil {
		t.Fatalf("Create room2 returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room1.ID); err != nil {
		t.Fatalf("Join room1 returned error: %v", err)
	}

	err = service.Join(testCtx, member.ID, room2.ID)
	if !errors.Is(err, ErrPlayerAlreadyInAnotherRoom) {
		t.Fatalf("Join error = %v, want %v", err, ErrPlayerAlreadyInAnotherRoom)
	}
}

func TestJoinSerializesConcurrentRoomMembership(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner1 := players.add("alice")
	owner2 := players.add("bob")
	member := players.add("carl")
	room1, err := service.Create(testCtx, owner1.ID, 0)
	if err != nil {
		t.Fatalf("Create room1 returned error: %v", err)
	}
	room2, err := service.Create(testCtx, owner2.ID, 0)
	if err != nil {
		t.Fatalf("Create room2 returned error: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, roomID := range []int64{room1.ID, room2.ID} {
		wg.Add(1)
		go func(roomID int64) {
			defer wg.Done()
			errs <- service.Join(testCtx, member.ID, roomID)
		}(roomID)
	}
	wg.Wait()
	close(errs)

	successes := 0
	conflicts := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPlayerAlreadyInAnotherRoom):
			conflicts++
		default:
			t.Fatalf("Join error = %v, want nil or %v", err, ErrPlayerAlreadyInAnotherRoom)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes = %d, conflicts = %d, want 1 and 1", successes, conflicts)
	}

	roomID, ok, err := rooms.FindRoomByPlayer(testCtx, member.ID)
	if err != nil {
		t.Fatalf("FindRoomByPlayer returned error: %v", err)
	}
	if !ok {
		t.Fatalf("member room index not found")
	}
	if roomID != room1.ID && roomID != room2.ID {
		t.Fatalf("member room id = %d, want %d or %d", roomID, room1.ID, room2.ID)
	}
}

func TestLeaveClearsPlayerRoomIndex(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	member := players.add("bob")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := service.Join(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}

	if err := service.Leave(testCtx, member.ID, room.ID); err != nil {
		t.Fatalf("Leave returned error: %v", err)
	}

	if roomID, ok, err := rooms.FindRoomByPlayer(testCtx, member.ID); err != nil {
		t.Fatalf("FindRoomByPlayer returned error: %v", err)
	} else if ok {
		t.Fatalf("member room index = %d, want cleared", roomID)
	}
}

func TestLeaveLastPlayerClearsPlayerRoomIndex(t *testing.T) {
	players, rooms, service := newTestRoomService()
	owner := players.add("alice")
	room, err := service.Create(testCtx, owner.ID, 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := service.Leave(testCtx, owner.ID, room.ID); err != nil {
		t.Fatalf("Leave returned error: %v", err)
	}

	if roomID, ok, err := rooms.FindRoomByPlayer(testCtx, owner.ID); err != nil {
		t.Fatalf("FindRoomByPlayer returned error: %v", err)
	} else if ok {
		t.Fatalf("owner room index = %d, want cleared", roomID)
	}
}

type testPlayerRepo struct {
	nextID  int64
	players map[int64]*player.Player
}

type testRoomRepo struct {
	nextID      int64
	rooms       map[int64]*Room
	playerRooms map[int64]int64
}

func newTestRoomService() (*testPlayerRepo, *testRoomRepo, *GameRoomService) {
	players := &testPlayerRepo{
		nextID:  1,
		players: make(map[int64]*player.Player),
	}
	rooms := &testRoomRepo{
		nextID:      1,
		rooms:       make(map[int64]*Room),
		playerRooms: make(map[int64]int64),
	}
	return players, rooms, NewService(players, rooms, DefaultConfig())
}

func (r *testPlayerRepo) add(name string) *player.Player {
	p := &player.Player{ID: r.nextID, Name: name}
	r.nextID++
	r.players[p.ID] = p
	return p
}

func (r *testPlayerRepo) NextID(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	id := r.nextID
	r.nextID++
	return id, nil
}

func (r *testPlayerRepo) Create(ctx context.Context, p *player.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.players[p.ID] = cloneTestPlayer(p)
	return nil
}

func (r *testPlayerRepo) Get(ctx context.Context, id int64) (*player.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, ok := r.players[id]
	if !ok {
		return nil, player.ErrNotFound
	}
	return cloneTestPlayer(p), nil
}

func (r *testPlayerRepo) Exists(ctx context.Context, id int64) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, ok := r.players[id]
	return ok, nil
}

func (r *testPlayerRepo) UpdateProfile(ctx context.Context, id int64, input player.UpdateProfileInput) (*player.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, ok := r.players[id]
	if !ok {
		return nil, player.ErrNotFound
	}
	if input.Name != nil {
		p.Name = *input.Name
	}
	if input.Avatar != nil {
		p.Avatar = *input.Avatar
	}
	if input.Email != nil {
		p.Email = *input.Email
	}
	if input.Phone != nil {
		p.Phone = *input.Phone
	}
	return cloneTestPlayer(p), nil
}

func cloneTestPlayer(p *player.Player) *player.Player {
	return &player.Player{
		ID:     p.ID,
		Name:   p.Name,
		Avatar: p.Avatar,
		Email:  p.Email,
		Phone:  p.Phone,
	}
}

func (r *testRoomRepo) NextID(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	id := r.nextID
	r.nextID++
	return id, nil
}

func (r *testRoomRepo) Create(ctx context.Context, room *Room) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.rooms[room.ID] = cloneTestRoom(room)
	return nil
}

func (r *testRoomRepo) CreateWithOwner(ctx context.Context, room *Room) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := r.playerRooms[room.OwnerID]; ok {
		return ErrPlayerAlreadyInAnotherRoom
	}
	r.rooms[room.ID] = cloneTestRoom(room)
	r.playerRooms[room.OwnerID] = room.ID
	return nil
}

func (r *testRoomRepo) Get(ctx context.Context, roomID int64) (*Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneTestRoom(room), nil
}

func (r *testRoomRepo) ListIDs(ctx context.Context) ([]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(r.rooms))
	for id := range r.rooms {
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *testRoomRepo) Exists(ctx context.Context, roomID int64) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, ok := r.rooms[roomID]
	return ok, nil
}

func (r *testRoomRepo) AddPlayer(ctx context.Context, roomID, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	if _, ok := room.Players[playerID]; ok {
		return ErrPlayerAlreadyInThisRoom
	}
	room.Players[playerID] = struct{}{}
	return nil
}

func (r *testRoomRepo) JoinRoom(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if currentRoomID, ok := r.playerRooms[playerID]; ok {
		if currentRoomID == roomID {
			return ErrPlayerAlreadyInThisRoom
		}
		return ErrPlayerAlreadyInAnotherRoom
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	if room.Status != StatusWaiting {
		return ErrRoomNotWaiting
	}
	if _, ok := room.Players[playerID]; ok {
		return ErrPlayerAlreadyInThisRoom
	}
	if len(room.Players) >= room.MaxPlayers {
		return ErrRoomFull
	}
	room.Players[playerID] = struct{}{}
	r.playerRooms[playerID] = roomID
	return nil
}

func (r *testRoomRepo) RemovePlayer(ctx context.Context, roomID, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	if _, ok := room.Players[playerID]; !ok {
		return ErrPlayerNotIn
	}
	delete(room.Players, playerID)
	return nil
}

func (r *testRoomRepo) LeaveRoom(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	if _, ok := room.Players[playerID]; !ok {
		return ErrPlayerNotIn
	}
	delete(room.Players, playerID)
	delete(room.ReadyPlayers, playerID)
	delete(r.playerRooms, playerID)

	if len(room.Players) == 0 {
		delete(r.rooms, roomID)
		return nil
	}
	if playerID == room.OwnerID {
		newOwnerID := minTestPlayerID(room.Players)
		room.OwnerID = newOwnerID
		delete(room.ReadyPlayers, newOwnerID)
	}
	return nil
}

func (r *testRoomRepo) RemoveReadyPlayer(ctx context.Context, roomID, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	delete(room.ReadyPlayers, playerID)
	return nil
}

func (r *testRoomRepo) AddReadyPlayer(ctx context.Context, roomID, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	room.ReadyPlayers[playerID] = struct{}{}
	return nil
}

func (r *testRoomRepo) SetReady(ctx context.Context, playerID, roomID int64, ready bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	if room.Status != StatusWaiting {
		return ErrRoomNotWaiting
	}
	if _, ok := room.Players[playerID]; !ok {
		return ErrPlayerNotIn
	}
	if playerID == room.OwnerID {
		return ErrOwnerCannotReadyOrUnready
	}
	if ready {
		room.ReadyPlayers[playerID] = struct{}{}
	} else {
		delete(room.ReadyPlayers, playerID)
	}
	return nil
}

func (r *testRoomRepo) UpdateOwner(ctx context.Context, roomID, ownerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	room.OwnerID = ownerID
	return nil
}

func (r *testRoomRepo) Delete(ctx context.Context, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(r.rooms, roomID)
	return nil
}

func (r *testRoomRepo) UpdateStatus(ctx context.Context, roomID int64, status Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	room.Status = status
	return nil
}

func (r *testRoomRepo) StartRoom(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	room, ok := r.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	if room.Status != StatusWaiting {
		return ErrRoomNotWaiting
	}
	if playerID != room.OwnerID {
		return ErrOnlyOwnerCanStart
	}
	for playerInRoomID := range room.Players {
		if playerInRoomID == playerID {
			continue
		}
		if _, ready := room.ReadyPlayers[playerInRoomID]; !ready {
			return ErrPlayersNotReady
		}
	}
	room.Status = StatusPlaying
	return nil
}

func (r *testRoomRepo) FindRoomByPlayer(ctx context.Context, playerID int64) (int64, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	roomID, ok := r.playerRooms[playerID]
	return roomID, ok, nil
}

func (r *testRoomRepo) SetPlayerRoom(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.playerRooms[playerID] = roomID
	return nil
}

func (r *testRoomRepo) ClearPlayerRoom(ctx context.Context, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(r.playerRooms, playerID)
	return nil
}

func cloneTestRoom(room *Room) *Room {
	return &Room{
		ID:           room.ID,
		OwnerID:      room.OwnerID,
		Status:       room.Status,
		MaxPlayers:   room.MaxPlayers,
		Players:      maps.Clone(room.Players),
		ReadyPlayers: maps.Clone(room.ReadyPlayers),
	}
}

func minTestPlayerID(players map[int64]struct{}) int64 {
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
