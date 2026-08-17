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
			BattleUDPAddr:  "127.0.0.1:7001",
		},
	}
	service := NewService(repo)

	result, err := service.Start(context.Background(), 7, "axe", true)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if repo.playerID != 7 {
		t.Fatalf("repo player id = %d, want 7", repo.playerID)
	}
	if repo.weapon != "axe" {
		t.Fatalf("repo weapon = %q, want axe", repo.weapon)
	}
	if !repo.solo {
		t.Fatal("repo solo = false, want true")
	}
	if result.RoomName != "room-1" {
		t.Fatalf("room name = %q, want room-1", result.RoomName)
	}
}

func TestServiceStartDefaultsEmptyWeaponToSword(t *testing.T) {
	repo := &fakeRepository{
		result: &rcenter.MatchResult{
			Status: rcenter.MatchStatusWaiting,
		},
	}
	service := NewService(repo)

	_, err := service.Start(context.Background(), 7, "", false)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if repo.weapon != "sword" {
		t.Fatalf("repo weapon = %q, want sword", repo.weapon)
	}
}

func TestServiceStartInvalidPlayer(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.Start(context.Background(), 0, "", false)
	if !errors.Is(err, rcenter.ErrInvalidPlayerID) {
		t.Fatalf("Start error = %v, want %v", err, rcenter.ErrInvalidPlayerID)
	}
}

func TestServiceStartCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(&fakeRepository{})

	_, err := service.Start(ctx, 7, "", false)
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

func TestServiceResume(t *testing.T) {
	repo := &fakeRepository{result: &rcenter.MatchResult{Status: rcenter.MatchStatusMatched, RoomName: "room-1"}}
	service := NewService(repo)

	result, err := service.Resume(context.Background(), 7)
	if err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}
	if repo.resumedPlayerID != 7 {
		t.Fatalf("repo resumed player id = %d, want 7", repo.resumedPlayerID)
	}
	if result.RoomName != "room-1" {
		t.Fatalf("room name = %q, want room-1", result.RoomName)
	}
}

func TestServiceResumeInvalidPlayer(t *testing.T) {
	_, err := NewService(&fakeRepository{}).Resume(context.Background(), 0)
	if !errors.Is(err, rcenter.ErrInvalidPlayerID) {
		t.Fatalf("Resume error = %v, want %v", err, rcenter.ErrInvalidPlayerID)
	}
}

type fakeRepository struct {
	playerID         int64
	weapon           string
	solo             bool
	canceledPlayerID int64
	resumedPlayerID  int64
	result           *rcenter.MatchResult
	err              error
}

func (f *fakeRepository) StartMatch(ctx context.Context, playerID int64, weapon string, solo bool) (*rcenter.MatchResult, error) {
	f.playerID = playerID
	f.weapon = weapon
	f.solo = solo
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeRepository) CancelMatch(ctx context.Context, playerID int64) error {
	f.canceledPlayerID = playerID
	return f.err
}

func (f *fakeRepository) ResumeMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	f.resumedPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}
