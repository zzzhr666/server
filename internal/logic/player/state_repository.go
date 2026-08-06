package player

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
)

type StateRepository struct {
	stateClient statecontract.Client
}

// NewStateRepository 使用 state-server 客户端创建玩家仓储。
func NewStateRepository(client statecontract.Client) *StateRepository {
	return &StateRepository{
		stateClient: client,
	}
}

// NextID 通过 state-server 分配下一个玩家 ID。
func (s *StateRepository) NextID(ctx context.Context) (int64, error) {
	id, err := s.stateClient.NextPlayerID(ctx)
	if err != nil {
		return 0, mapStateError(err)
	}
	return id, nil
}

// Create 通过 state-server 持久化玩家。
func (s *StateRepository) Create(ctx context.Context, player *Player) error {
	return mapStateError(s.stateClient.CreatePlayer(ctx, toStatePlayer(player)))
}

// Get 通过 state-server 读取玩家。
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
		Coins:    player.Coins,
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
		Coins:    player.Coins,
	}
}

func mapStateError(err error) error {
	if errors.Is(err, statecontract.ErrPlayerNotFound) {
		return ErrNotFound
	}
	return err
}
