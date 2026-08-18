package leaderboard

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
)

func TestStateRepositoryListConvertsRequestAndResult(t *testing.T) {
	client := &fakeLeaderboardClient{
		result: &statecontract.ListLeaderboardResult{
			Type:       statecontract.LeaderboardTypeDuoClearTime,
			MapVersion: "wave-v1",
			Entries: []statecontract.LeaderboardEntry{
				{
					Rank:  1,
					Score: 82_000,
					Players: []statecontract.LeaderboardPlayer{
						{PlayerID: 7, Nickname: "Alice", Avatar: "mage"},
						{PlayerID: 8, Nickname: "Bob", Avatar: "knight"},
					},
				},
			},
		},
	}
	repository := NewStateRepository(client)

	result, err := repository.List(context.Background(), ListInput{
		Type:       TypeDuoClearTime,
		MapVersion: "wave-v1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if client.input.Type != statecontract.LeaderboardTypeDuoClearTime || client.input.MapVersion != "wave-v1" || client.input.Limit != 10 {
		t.Fatalf("state input = %+v, want duo wave-v1 limit 10", client.input)
	}
	if result.Type != TypeDuoClearTime || result.MapVersion != "wave-v1" || len(result.Entries) != 1 {
		t.Fatalf("result = %+v, want one converted duo entry", result)
	}
	entry := result.Entries[0]
	if entry.Rank != 1 || entry.Score != 82_000 || len(entry.Players) != 2 {
		t.Fatalf("entry = %+v, want rank 1 score 82000 and two players", entry)
	}
	if entry.Players[0].Nickname != "Alice" || entry.Players[0].Avatar != "mage" || entry.Players[1].PlayerID != 8 {
		t.Fatalf("players = %+v, want complete converted players", entry.Players)
	}
}

func TestStateRepositoryListMapsErrors(t *testing.T) {
	client := &fakeLeaderboardClient{err: statecontract.ErrInvalidLeaderboardQuery}
	_, err := NewStateRepository(client).List(context.Background(), ListInput{})
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("invalid query error = %v, want %v", err, ErrInvalidQuery)
	}

	wantErr := errors.New("state unavailable")
	client.err = wantErr
	_, err = NewStateRepository(client).List(context.Background(), ListInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("state error = %v, want %v", err, wantErr)
	}
}

func TestLeaderboardStateConversionsHandleUnknownAndNil(t *testing.T) {
	if got := toStateType("unknown"); got != "" {
		t.Fatalf("toStateType(unknown) = %q, want empty", got)
	}
	if got := fromStateType("unknown"); got != "" {
		t.Fatalf("fromStateType(unknown) = %q, want empty", got)
	}
	if got := fromStateResult(nil); got != nil {
		t.Fatalf("fromStateResult(nil) = %+v, want nil", got)
	}
}

type fakeLeaderboardClient struct {
	input  statecontract.ListLeaderboardInput
	result *statecontract.ListLeaderboardResult
	err    error
}

func (f *fakeLeaderboardClient) ListLeaderboard(_ context.Context, input statecontract.ListLeaderboardInput) (*statecontract.ListLeaderboardResult, error) {
	f.input = input
	return f.result, f.err
}

var _ statecontract.LeaderboardClient = (*fakeLeaderboardClient)(nil)
