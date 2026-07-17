package player

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
)

type StateRepository struct {
	stateClient statecontract.Client
}

// NewStateRepository creates a player repository backed by a state-server client.
func NewStateRepository(client statecontract.Client) *StateRepository {
	return &StateRepository{
		stateClient: client,
	}
}

// NextID allocates the next player ID through state-server.
func (s *StateRepository) NextID(ctx context.Context) (int64, error) {
	id, err := s.stateClient.NextPlayerID(ctx)
	if err != nil {
		return 0, mapStateError(err)
	}
	return id, nil
}

// Create persists a player through state-server.
func (s *StateRepository) Create(ctx context.Context, player *Player) error {
	return mapStateError(s.stateClient.CreatePlayer(ctx, toStatePlayer(player)))
}

// Get loads a player through state-server.
func (s *StateRepository) Get(ctx context.Context, id int64) (*Player, error) {
	player, err := s.stateClient.GetPlayer(ctx, id)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStatePlayer(player), nil
}

func toStatePlayer(player *Player) *statecontract.Player {
	if player == nil {
		return nil
	}
	return &statecontract.Player{
		ID:       player.ID,
		Nickname: player.Nickname,
		Avatar:   player.Avatar,
		Email:    player.Email,
		Phone:    player.Phone,
	}
}
func fromStatePlayer(player *statecontract.Player) *Player {
	if player == nil {
		return nil
	}
	return &Player{
		ID:       player.ID,
		Nickname: player.Nickname,
		Avatar:   player.Avatar,
		Email:    player.Email,
		Phone:    player.Phone,
	}
}

func mapStateError(err error) error {
	if errors.Is(err, statecontract.ErrPlayerNotFound) {
		return ErrNotFound
	}
	return err
}
