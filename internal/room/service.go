package room

import (
	"context"
	"learning/internal/player"
)

type Service interface {
	Create(ctx context.Context, ownerID int64) (*Room, error)
	Get(ctx context.Context, roomID int64) (*Room, error)
	List(ctx context.Context) ([]*Room, error)
	Join(ctx context.Context, playerID, roomID int64) error
	Leave(ctx context.Context, playerID, roomID int64) error
}

type Repository interface {
	NextID(ctx context.Context) (int64, error)
	Create(ctx context.Context, r *Room) error
	Get(ctx context.Context, roomID int64) (*Room, error)
	ListIDs(ctx context.Context) ([]int64, error)
	Exists(ctx context.Context, roomID int64) (bool, error)
	AddPlayer(ctx context.Context, roomID, playerID int64) error
	RemovePlayer(ctx context.Context, roomID, playerID int64) error
}

type RoomService struct {
	players player.Repository
	rooms   Repository
}

func NewService(players player.Repository, rooms Repository) *RoomService {
	return &RoomService{players: players, rooms: rooms}
}

func (s *RoomService) Create(ctx context.Context, ownerID int64) (*Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ok, err := s.players.Exists(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPlayerNotFound
	}
	roomID, err := s.rooms.NextID(ctx)
	if err != nil {
		return nil, err
	}
	r := &Room{ID: roomID, OwnerID: ownerID, Players: map[int64]struct{}{ownerID: {}}}
	if err := s.rooms.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *RoomService) Get(ctx context.Context, roomID int64) (*Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.rooms.Get(ctx, roomID)
}

func (s *RoomService) List(ctx context.Context) ([]*Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids, err := s.rooms.ListIDs(ctx)
	if err != nil {
		return nil, err
	}
	rooms := make([]*Room, 0, len(ids))
	for _, id := range ids {
		r, err := s.rooms.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, nil
}

func (s *RoomService) Join(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.players.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	ok, err = s.rooms.Exists(ctx, roomID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return s.rooms.AddPlayer(ctx, roomID, playerID)
}

func (s *RoomService) Leave(ctx context.Context, playerID, roomID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ok, err := s.players.Exists(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlayerNotFound
	}
	ok, err = s.rooms.Exists(ctx, roomID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return s.rooms.RemovePlayer(ctx, roomID, playerID)
}
