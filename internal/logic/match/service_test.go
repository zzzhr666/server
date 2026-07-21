package match

import (
	"context"
	"errors"
	"testing"

	"server/internal/rcenter"
)

func TestServiceStart(t *testing.T) {
	repo := &fakeRepository{
		result: &rcenter.MatchResult{
			Status:         rcenter.MatchStatusMatched,
			RoomName:       "room-1",
			Token:          "token-1",
			BattleNodeName: "battle-1",
			BattleKCPAddr:  "127.0.0.1:7001",
		},
	}
	service := NewService(repo)

	result, err := service.Start(context.Background(), 7)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if repo.playerID != 7 {
		t.Fatalf("repo player id = %d, want 7", repo.playerID)
	}
	if result.RoomName != "room-1" {
		t.Fatalf("room name = %q, want room-1", result.RoomName)
	}
}

func TestServiceStartInvalidPlayer(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.Start(context.Background(), 0)
	if !errors.Is(err, rcenter.ErrInvalidPlayerID) {
		t.Fatalf("Start error = %v, want %v", err, rcenter.ErrInvalidPlayerID)
	}
}

func TestServiceStartCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(&fakeRepository{})

	_, err := service.Start(ctx, 7)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start error = %v, want %v", err, context.Canceled)
	}
}

func TestServiceCancel(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	err := service.Cancel(context.Background(), 7)
	if err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if repo.canceledPlayerID != 7 {
		t.Fatalf("repo canceled player id = %d, want 7", repo.canceledPlayerID)
	}
}

func TestServiceCancelInvalidPlayer(t *testing.T) {
	service := NewService(&fakeRepository{})

	err := service.Cancel(context.Background(), 0)
	if !errors.Is(err, rcenter.ErrInvalidPlayerID) {
		t.Fatalf("Cancel error = %v, want %v", err, rcenter.ErrInvalidPlayerID)
	}
}

func TestServiceCancelCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(&fakeRepository{})

	err := service.Cancel(ctx, 7)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Cancel error = %v, want %v", err, context.Canceled)
	}
}

type fakeRepository struct {
	playerID         int64
	canceledPlayerID int64
	result           *rcenter.MatchResult
	err              error
}

func (f *fakeRepository) StartMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	f.playerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeRepository) CancelMatch(ctx context.Context, playerID int64) error {
	f.canceledPlayerID = playerID
	return f.err
}
