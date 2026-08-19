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

	result, err := repo.StartMatch(context.Background(), 7, "ice", true)
	if err != nil {
		t.Fatalf("StartMatch returned error: %v", err)
	}
	if client.playerID != 7 {
		t.Fatalf("client player id = %d, want 7", client.playerID)
	}
	if client.hero != "ice" {
		t.Fatalf("client hero = %q, want ice", client.hero)
	}
	if !client.solo {
		t.Fatal("client solo = false, want true")
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

func TestRCenterRepositoryResumeMatch(t *testing.T) {
	client := &fakeRCenterClient{result: &rcenter.MatchResult{Status: rcenter.MatchStatusMatched, RoomName: "room-1"}}
	repo := NewRCenterRepository(client)

	result, err := repo.ResumeMatch(context.Background(), 7)
	if err != nil {
		t.Fatalf("ResumeMatch returned error: %v", err)
	}
	if client.resumedPlayerID != 7 {
		t.Fatalf("client resumed player id = %d, want 7", client.resumedPlayerID)
	}
	if result.RoomName != "room-1" {
		t.Fatalf("room name = %q, want room-1", result.RoomName)
	}
}

type fakeRCenterClient struct {
	playerID         int64
	hero             string
	solo             bool
	canceledPlayerID int64
	resumedPlayerID  int64
	result           *rcenter.MatchResult
	err              error
}

func (f *fakeRCenterClient) StartMatch(ctx context.Context, playerID int64, hero string, solo bool) (*rcenter.MatchResult, error) {
	f.playerID = playerID
	f.hero = hero
	f.solo = solo
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeRCenterClient) CancelMatch(ctx context.Context, playerID int64) error {
	f.canceledPlayerID = playerID
	return f.err
}

func (f *fakeRCenterClient) ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	f.resumedPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}
