package match

import (
	"context"
	"testing"

	"server/internal/rcenter"
)

func TestRCenterRepositoryStartMatch(t *testing.T) {
	client := &fakeRCenterClient{
		result: &rcenter.MatchResult{
			Status:   rcenter.MatchStatusWaiting,
			RoomName: "",
		},
	}
	repo := NewRCenterRepository(client)

	result, err := repo.StartMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("StartMatch returned error: %v", err)
	}
	if client.playerID != 7 {
		t.Fatalf("client player id = %d, want 7", client.playerID)
	}
	if result.Status != rcenter.MatchStatusWaiting {
		t.Fatalf("status = %q, want %q", result.Status, rcenter.MatchStatusWaiting)
	}
}

func TestRCenterRepositoryCancelMatch(t *testing.T) {
	client := &fakeRCenterClient{}
	repo := NewRCenterRepository(client)

	err := repo.CancelMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}
	if client.canceledPlayerID != 7 {
		t.Fatalf("client canceled player id = %d, want 7", client.canceledPlayerID)
	}
}

type fakeRCenterClient struct {
	playerID         int64
	canceledPlayerID int64
	result           *rcenter.MatchResult
	err              error
}

func (f *fakeRCenterClient) StartMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	f.playerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeRCenterClient) CancelMatch(ctx context.Context, playerID int64) error {
	f.canceledPlayerID = playerID
	return f.err
}
