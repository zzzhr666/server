package presence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceMarkOnline(t *testing.T) {
	repo := &fakePresenceRepository{}
	svc := NewService(repo)

	if err := svc.MarkOnline(context.Background(), 7, "logic-1"); err != nil {
		t.Fatalf("MarkOnline returned error: %v", err)
	}
	if repo.setPresence.PlayerID != 7 {
		t.Fatalf("player id = %d, want 7", repo.setPresence.PlayerID)
	}
	if repo.setPresence.ServerName != "logic-1" {
		t.Fatalf("server name = %q, want logic-1", repo.setPresence.ServerName)
	}
	if repo.setPresence.Status != StatusOnline {
		t.Fatalf("status = %q, want %q", repo.setPresence.Status, StatusOnline)
	}
	if repo.setPresence.UpdatedAt.IsZero() {
		t.Fatalf("updated at is zero, want current time")
	}
	if repo.setTTL != DefaultTTL {
		t.Fatalf("ttl = %v, want %v", repo.setTTL, DefaultTTL)
	}
}

func TestServiceMarkOnlineInvalidInput(t *testing.T) {
	repo := &fakePresenceRepository{}
	svc := NewService(repo)

	err := svc.MarkOnline(context.Background(), 0, "logic-1")
	if !errors.Is(err, ErrInvalidPresence) {
		t.Fatalf("MarkOnline error = %v, want %v", err, ErrInvalidPresence)
	}
	if repo.setPresence != nil {
		t.Fatalf("set presence = %+v, want nil", repo.setPresence)
	}
}

func TestServiceGet(t *testing.T) {
	updatedAt := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	repo := &fakePresenceRepository{
		presence: &Presence{
			PlayerID:   7,
			ServerName: "logic-1",
			Status:     StatusOnline,
			UpdatedAt:  updatedAt,
		},
	}
	svc := NewService(repo)

	got, err := svc.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.PlayerID != 7 {
		t.Fatalf("player id = %d, want 7", got.PlayerID)
	}
	if got.ServerName != "logic-1" {
		t.Fatalf("server name = %q, want logic-1", got.ServerName)
	}
	if repo.gotPlayerID != 7 {
		t.Fatalf("repo got player id = %d, want 7", repo.gotPlayerID)
	}
}

func TestServiceGetInvalidInput(t *testing.T) {
	svc := NewService(&fakePresenceRepository{})

	_, err := svc.Get(context.Background(), 0)
	if !errors.Is(err, ErrInvalidPresence) {
		t.Fatalf("Get error = %v, want %v", err, ErrInvalidPresence)
	}
}

func TestServiceMarkOffline(t *testing.T) {
	repo := &fakePresenceRepository{}
	svc := NewService(repo)

	if err := svc.MarkOffline(context.Background(), 7, "logic-1"); err != nil {
		t.Fatalf("MarkOffline returned error: %v", err)
	}
	if repo.clearedPlayerID != 7 {
		t.Fatalf("cleared player id = %d, want 7", repo.clearedPlayerID)
	}
	if repo.clearedServerName != "logic-1" {
		t.Fatalf("cleared server name = %q, want logic-1", repo.clearedServerName)
	}
}

func TestServiceMarkOfflineInvalidInput(t *testing.T) {
	repo := &fakePresenceRepository{}
	svc := NewService(repo)

	err := svc.MarkOffline(context.Background(), 7, "")
	if !errors.Is(err, ErrInvalidPresence) {
		t.Fatalf("MarkOffline error = %v, want %v", err, ErrInvalidPresence)
	}
	if repo.clearedPlayerID != 0 {
		t.Fatalf("cleared player id = %d, want 0", repo.clearedPlayerID)
	}
}

func TestServiceRefresh(t *testing.T) {
	repo := &fakePresenceRepository{}
	svc := NewService(repo)

	if err := svc.Refresh(context.Background(), 7, "logic-1"); err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if repo.refreshedPlayerID != 7 {
		t.Fatalf("refreshed player id = %d, want 7", repo.refreshedPlayerID)
	}
	if repo.refreshedServerName != "logic-1" {
		t.Fatalf("refreshed server name = %q, want logic-1", repo.refreshedServerName)
	}
	if repo.refreshedAt.IsZero() {
		t.Fatalf("refreshed at is zero, want current time")
	}
	if repo.refreshedTTL != DefaultTTL {
		t.Fatalf("refreshed ttl = %v, want %v", repo.refreshedTTL, DefaultTTL)
	}
}

func TestServiceRefreshInvalidInput(t *testing.T) {
	repo := &fakePresenceRepository{}
	svc := NewService(repo)

	err := svc.Refresh(context.Background(), 0, "logic-1")
	if !errors.Is(err, ErrInvalidPresence) {
		t.Fatalf("Refresh error = %v, want %v", err, ErrInvalidPresence)
	}
	if repo.refreshedPlayerID != 0 {
		t.Fatalf("refreshed player id = %d, want 0", repo.refreshedPlayerID)
	}
}

type fakePresenceRepository struct {
	presence            *Presence
	setPresence         *Presence
	setTTL              time.Duration
	gotPlayerID         int64
	clearedPlayerID     int64
	clearedServerName   string
	refreshedPlayerID   int64
	refreshedServerName string
	refreshedAt         time.Time
	refreshedTTL        time.Duration
	err                 error
}

func (f *fakePresenceRepository) SetPresence(_ context.Context, presence *Presence, ttl time.Duration) error {
	f.setPresence = presence
	f.setTTL = ttl
	return f.err
}

func (f *fakePresenceRepository) GetPresence(_ context.Context, playerID int64) (*Presence, error) {
	f.gotPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.presence, nil
}

func (f *fakePresenceRepository) ClearPresence(_ context.Context, playerID int64, serverName string) error {
	f.clearedPlayerID = playerID
	f.clearedServerName = serverName
	return f.err
}

func (f *fakePresenceRepository) RefreshPresence(_ context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	f.refreshedPlayerID = playerID
	f.refreshedServerName = serverName
	f.refreshedAt = updatedAt
	f.refreshedTTL = ttl
	return f.err
}

var _ Repository = (*fakePresenceRepository)(nil)
