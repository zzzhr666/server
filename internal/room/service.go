package room

import (
	"context"
	"learning/internal/player"
	"sync"
)

// Service defines room business operations.
type Service interface {
	// Create creates a waiting room and adds the owner as the first player.
	Create(ctx context.Context, ownerID int64, maxPlayers int) (*Room, error)
	// Get returns a room by ID.
	Get(ctx context.Context, roomID int64) (*Room, error)
	// List returns all rooms.
	List(ctx context.Context) ([]*Room, error)
	// Join adds a player to a waiting room if the room has capacity.
	Join(ctx context.Context, playerID, roomID int64) error
	// Leave removes a player from a room and handles owner transfer or room deletion.
	Leave(ctx context.Context, playerID, roomID int64) error
	// Ready marks a non-owner room player as ready.
	Ready(ctx context.Context, playerID, roomID int64) error
	// Unready removes a non-owner room player from the ready set.
	Unready(ctx context.Context, playerID, roomID int64) error
	// Start moves a waiting room to playing after owner and readiness checks.
	Start(ctx context.Context, playerID, roomID int64) error
}

// Repository defines room persistence operations used by the service layer.
type Repository interface {
	// NextID allocates the next room ID.
	NextID(ctx context.Context) (int64, error)
	// Create persists a room and its player sets.
	Create(ctx context.Context, r *Room) error
	// Get loads a room by ID.
	Get(ctx context.Context, roomID int64) (*Room, error)
	// ListIDs returns all room IDs.
	ListIDs(ctx context.Context) ([]int64, error)
	// Exists reports whether a room exists.
	Exists(ctx context.Context, roomID int64) (bool, error)
	// AddPlayer adds a player to the room member set.
	AddPlayer(ctx context.Context, roomID, playerID int64) error
	// RemovePlayer removes a player from the room member set.
	RemovePlayer(ctx context.Context, roomID, playerID int64) error
	// RemoveReadyPlayer removes a player from the room ready set.
	RemoveReadyPlayer(ctx context.Context, roomID, playerID int64) error
	// AddReadyPlayer adds a player to the room ready set.
	AddReadyPlayer(ctx context.Context, roomID, playerID int64) error
	// UpdateOwner updates the room owner ID.
	UpdateOwner(ctx context.Context, roomID, ownerID int64) error
	// Delete removes a room and its related Redis data.
	Delete(ctx context.Context, roomID int64) error
	// UpdateStatus updates the room status.
	UpdateStatus(ctx context.Context, roomID int64, status Status) error
	// FindRoomByPlayer returns the room currently indexed for a player.
	FindRoomByPlayer(ctx context.Context, playerID int64) (int64, bool, error)
	// SetPlayerRoom indexes the current room for a player.
	SetPlayerRoom(ctx context.Context, playerID, roomID int64) error
	// ClearPlayerRoom removes the current room index for a player.
	ClearPlayerRoom(ctx context.Context, playerID int64) error
}

// Config contains room service limits.
type Config struct {
	MaxPlayers int
}

// DefaultConfig returns the default room service configuration.
func DefaultConfig() Config {
	return Config{MaxPlayers: 10}
}

// GameRoomService implements room business rules.
type GameRoomService struct {
	playersRepo player.Repository
	roomsRepo   Repository
	config      Config
	mu          sync.RWMutex
}

// NewService creates a room service with player and room repositories.
func NewService(playersRepo player.Repository, roomsRepo Repository, config Config) *GameRoomService {
	return &GameRoomService{playersRepo: playersRepo, roomsRepo: roomsRepo, config: config}
}

// Create creates a waiting room, validates the owner, and applies max player limits.
func (s *GameRoomService) Create(ctx context.Context, ownerID int64, maxPlayers int) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ok, err := s.playersRepo.Exists(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPlayerNotFound
	}
	if maxPlayers == 0 {
		maxPlayers = s.config.MaxPlayers
	}
	if maxPlayers < 1 || maxPlayers > s.config.MaxPlayers {
		return nil, ErrInvalidMaxPlayers
	}
	_, ok, err = s.roomsRepo.FindRoomByPlayer(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, ErrPlayerAlreadyInAnotherRoom
	}
	roomID, err := s.roomsRepo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	r := &Room{
		ID:           roomID,
		OwnerID:      ownerID,
		Status:       StatusWaiting,
		MaxPlayers:   maxPlayers,
		Players:      map[int64]struct{}{ownerID: {}},
		ReadyPlayers: make(map[int64]struct{}),
	}
	if err := s.roomsRepo.Create(ctx, r); err != nil {
		return nil, err
	}
	if err := s.roomsRepo.SetPlayerRoom(ctx, ownerID, roomID); err != nil {
		return nil, err
	}
	return r, nil
}

// Get returns a room by ID.
func (s *GameRoomService) Get(ctx context.Context, roomID int64) (*Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.roomsRepo.Get(ctx, roomID)
}

// List returns all rooms by loading each stored room ID.
func (s *GameRoomService) List(ctx context.Context) ([]*Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids, err := s.roomsRepo.ListIDs(ctx)
	if err != nil {
		return nil, err
	}
	rooms := make([]*Room, 0, len(ids))
	for _, id := range ids {
		r, err := s.roomsRepo.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, nil
}

// Join adds an existing player to a waiting room when capacity allows.
func (s *GameRoomService) Join(ctx context.Context, playerID, roomID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.playersRepo.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	_, ok, err = s.roomsRepo.FindRoomByPlayer(ctx, playerID)
	if err != nil {
		return err
	}
	if ok {
		return ErrPlayerAlreadyInAnotherRoom
	}
	room, err := s.roomsRepo.Get(ctx, roomID)
	if err != nil {
		return err
	}
	if room.Status != StatusWaiting {
		return ErrRoomNotWaiting
	}

	if len(room.Players) >= room.MaxPlayers {
		return ErrRoomFull
	}

	if err := s.roomsRepo.AddPlayer(ctx, roomID, playerID); err != nil {
		return err
	}
	return s.roomsRepo.SetPlayerRoom(ctx, playerID, roomID)
}

// Leave removes a player, clears readiness, transfers ownership, or deletes an empty room.
func (s *GameRoomService) Leave(ctx context.Context, playerID, roomID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.playersRepo.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	room, err := s.roomsRepo.Get(ctx, roomID)
	if err != nil {
		return err
	}

	if _, ok := room.Players[playerID]; !ok {
		return ErrPlayerNotIn
	}
	delete(room.Players, playerID)

	if err := s.roomsRepo.RemovePlayer(ctx, roomID, playerID); err != nil {
		return err
	}
	if err := s.roomsRepo.RemoveReadyPlayer(ctx, roomID, playerID); err != nil {
		return err
	}
	if err := s.roomsRepo.ClearPlayerRoom(ctx, playerID); err != nil {
		return err
	}

	if len(room.Players) == 0 {
		return s.roomsRepo.Delete(ctx, roomID)
	}

	if playerID == room.OwnerID {
		newOwnerID := minPlayerID(room.Players)
		err := s.roomsRepo.UpdateOwner(ctx, roomID, newOwnerID)
		if err != nil {
			return err
		}
		err = s.roomsRepo.RemoveReadyPlayer(ctx, roomID, newOwnerID)
		if err != nil {
			return err
		}
	}
	return nil
}

// Unready removes a non-owner room player from the ready set.
func (s *GameRoomService) Unready(ctx context.Context, playerID, roomID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.playersRepo.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	room, err := s.roomsRepo.Get(ctx, roomID)
	if err != nil {
		return err
	}
	if room.Status != StatusWaiting {
		return ErrRoomNotWaiting
	}
	_, inRoom := room.Players[playerID]
	if !inRoom {
		return ErrPlayerNotIn
	}
	if playerID == room.OwnerID {
		return ErrOwnerCannotReadyOrUnready
	}
	return s.roomsRepo.RemoveReadyPlayer(ctx, roomID, playerID)
}

// Ready marks a non-owner room player as ready while the room is waiting.
func (s *GameRoomService) Ready(ctx context.Context, playerID, roomID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.playersRepo.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	room, err := s.roomsRepo.Get(ctx, roomID)
	if err != nil {
		return err
	}
	if room.Status != StatusWaiting {
		return ErrRoomNotWaiting
	}
	_, inRoom := room.Players[playerID]
	if !inRoom {
		return ErrPlayerNotIn
	}
	if playerID == room.OwnerID {
		return ErrOwnerCannotReadyOrUnready
	}
	return s.roomsRepo.AddReadyPlayer(ctx, roomID, playerID)
}

// minPlayerID returns the smallest player ID from a non-empty player set.
func minPlayerID(players map[int64]struct{}) int64 {
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

// Start moves a waiting room to playing when the owner starts and all members are ready.
func (s *GameRoomService) Start(ctx context.Context, playerID, roomID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.playersRepo.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	room, err := s.roomsRepo.Get(ctx, roomID)
	if err != nil {
		return err
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
		_, ready := room.ReadyPlayers[playerInRoomID]
		if !ready {
			return ErrPlayersNotReady
		}
	}

	return s.roomsRepo.UpdateStatus(ctx, roomID, StatusPlaying)
}
