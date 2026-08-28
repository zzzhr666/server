package leaderboard

import (
	"context"
	"errors"
	"server/internal/contract/state"
)

// StateRepository 将排行榜查询适配到 state-server。
type StateRepository struct {
	client state.LeaderboardClient
}

// List 从 state-server 读取排行榜并转换为 logic 领域模型。
func (s *StateRepository) List(ctx context.Context, input ListInput) (*Result, error) {
	stateInput := state.ListLeaderboardInput{
		Type:       toStateType(input.Type),
		MapVersion: input.MapVersion,
		Limit:      input.Limit,
	}
	res, err := s.client.ListLeaderboard(ctx, stateInput)
	if err != nil {
		if errors.Is(err, state.ErrInvalidLeaderboardQuery) {
			return nil, ErrInvalidQuery
		}
		return nil, err
	}
	return fromStateResult(res), nil
}

// NewStateRepository 使用 state-server 排行榜客户端创建仓储。
func NewStateRepository(client state.LeaderboardClient) *StateRepository {
	return &StateRepository{client: client}
}

func toStateType(t Type) state.LeaderboardType {
	switch t {
	case TypeTotalKills:
		return state.LeaderboardTypeTotalKills
	case TypeSoloClearTime:
		return state.LeaderboardTypeSoloClearTime
	case TypeDuoClearTime:
		return state.LeaderboardTypeDuoClearTime
	case TypeTrioClearTime:
		return state.LeaderboardTypeTrioClearTime
	case TypeQuadClearTime:
		return state.LeaderboardTypeQuadClearTime
	default:
		return ""
	}
}

func fromStateType(t state.LeaderboardType) Type {
	switch t {
	case state.LeaderboardTypeTotalKills:
		return TypeTotalKills
	case state.LeaderboardTypeSoloClearTime:
		return TypeSoloClearTime
	case state.LeaderboardTypeDuoClearTime:
		return TypeDuoClearTime
	case state.LeaderboardTypeTrioClearTime:
		return TypeTrioClearTime
	case state.LeaderboardTypeQuadClearTime:
		return TypeQuadClearTime
	default:
		return ""
	}
}

func fromStateResult(result *state.ListLeaderboardResult) *Result {
	if result == nil {
		return nil
	}
	entries := make([]Entry, 0, len(result.Entries))
	for i := range result.Entries {
		entry := &result.Entries[i]
		players := make([]Player, 0, len(entry.Players))
		for j := range entry.Players {
			players = append(players, Player{
				PlayerID: entry.Players[j].PlayerID,
				Nickname: entry.Players[j].Nickname,
				Avatar:   entry.Players[j].Avatar,
			})
		}
		entries = append(entries, Entry{
			Rank:    entry.Rank,
			Players: players,
			Score:   entry.Score,
		})
	}
	return &Result{
		Type:       fromStateType(result.Type),
		MapVersion: result.MapVersion,
		Entries:    entries,
	}
}

var _ Repository = (*StateRepository)(nil)
