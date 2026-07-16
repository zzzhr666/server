package room

import (
	"context"
	"learning/internal/player"
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
	// CreateWithOwner persists a room and indexes the owner in one transaction.
	CreateWithOwner(ctx context.Context, r *Room) error
	// JoinRoom adds a player to a waiting room and indexes membership atomically.
	JoinRoom(ctx context.Context, playerID, roomID int64) error
	// LeaveRoom removes a player and applies owner transfer or room deletion atomically.
	LeaveRoom(ctx context.Context, playerID, roomID int64) error
	// SetReady updates a room player's ready state atomically.
	SetReady(ctx context.Context, playerID, roomID int64, ready bool) error
	// StartRoom marks a waiting room as playing after owner and readiness checks.
	StartRoom(ctx context.Context, playerID, roomID int64) error
	// FindRoomByPlayer returns the room currently indexed for a player.
	FindRoomByPlayer(ctx context.Context, playerID int64) (int64, bool, error)
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
}

// NewService creates a room service with player and room repositories.
func NewService(playersRepo player.Repository, roomsRepo Repository, config Config) *GameRoomService {
	return &GameRoomService{playersRepo: playersRepo, roomsRepo: roomsRepo, config: config}
}

// Create creates a waiting room, validates the owner, and applies max player limits.
func (s *GameRoomService) Create(ctx context.Context, ownerID int64, maxPlayers int) (*Room, error) {
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
	if err := s.roomsRepo.CreateWithOwner(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// Get returns a room by ID.
func (s *GameRoomService) Get(ctx context.Context, roomID int64) (*Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.roomsRepo.Get(ctx, roomID)
}

// List returns all rooms by loading each stored room ID.
func (s *GameRoomService) List(ctx context.Context) ([]*Room, error) {
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
	return s.roomsRepo.JoinRoom(ctx, playerID, roomID)
}

// Leave removes a player, clears readiness, transfers ownership, or deletes an empty room.
func (s *GameRoomService) Leave(ctx context.Context, playerID, roomID int64) error {
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
	return s.roomsRepo.LeaveRoom(ctx, playerID, roomID)
}

// Unready removes a non-owner room player from the ready set.
func (s *GameRoomService) Unready(ctx context.Context, playerID, roomID int64) error {
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
	return s.roomsRepo.SetReady(ctx, playerID, roomID, false)
}

// Ready marks a non-owner room player as ready while the room is waiting.
func (s *GameRoomService) Ready(ctx context.Context, playerID, roomID int64) error {
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
	return s.roomsRepo.SetReady(ctx, playerID, roomID, true)
}

// Start moves a waiting room to playing when the owner starts and all members are ready.
func (s *GameRoomService) Start(ctx context.Context, playerID, roomID int64) error {
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
	return s.roomsRepo.StartRoom(ctx, playerID, roomID)
}
