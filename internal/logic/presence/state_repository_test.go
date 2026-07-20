package presence

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestStateRepositorySetPresence(t *testing.T) {
	client := &fakeStateClient{}
	repo := NewStateRepository(client)
	updatedAt := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)

	err := repo.SetPresence(context.Background(), &Presence{
		PlayerID:   7,
		ServerName: "logic-1",
		Status:     StatusOnline,
		UpdatedAt:  updatedAt,
	}, time.Minute)
	if err != nil {
		t.Fatalf("SetPresence returned error: %v", err)
	}
	if client.setPresence.PlayerID != 7 {
		t.Fatalf("player id = %d, want 7", client.setPresence.PlayerID)
	}
	if client.setPresence.ServerName != "logic-1" {
		t.Fatalf("server name = %q, want logic-1", client.setPresence.ServerName)
	}
	if client.setPresence.Status != StatusOnline {
		t.Fatalf("status = %q, want %q", client.setPresence.Status, StatusOnline)
	}
	if client.setTTL != time.Minute {
		t.Fatalf("ttl = %v, want %v", client.setTTL, time.Minute)
	}
}

func TestStateRepositoryGetPresence(t *testing.T) {
	updatedAt := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	client := &fakeStateClient{
		presence: &statecontract.Presence{
			PlayerID:   7,
			ServerName: "logic-1",
			Status:     StatusOnline,
			UpdatedAt:  updatedAt,
		},
	}
	repo := NewStateRepository(client)

	got, err := repo.GetPresence(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetPresence returned error: %v", err)
	}
	if got.PlayerID != 7 {
		t.Fatalf("player id = %d, want 7", got.PlayerID)
	}
	if got.ServerName != "logic-1" {
		t.Fatalf("server name = %q, want logic-1", got.ServerName)
	}
	if !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, updatedAt)
	}
}

func TestStateRepositoryClearPresence(t *testing.T) {
	client := &fakeStateClient{}
	repo := NewStateRepository(client)

	if err := repo.ClearPresence(context.Background(), 7, "logic-1"); err != nil {
		t.Fatalf("ClearPresence returned error: %v", err)
	}
	if client.clearedPlayerID != 7 {
		t.Fatalf("cleared player id = %d, want 7", client.clearedPlayerID)
	}
	if client.clearedServerName != "logic-1" {
		t.Fatalf("cleared server name = %q, want logic-1", client.clearedServerName)
	}
}

func TestStateRepositoryRefreshPresence(t *testing.T) {
	client := &fakeStateClient{}
	repo := NewStateRepository(client)
	updatedAt := time.Date(2026, 7, 19, 14, 1, 0, 0, time.UTC)

	if err := repo.RefreshPresence(context.Background(), 7, "logic-1", updatedAt, time.Minute); err != nil {
		t.Fatalf("RefreshPresence returned error: %v", err)
	}
	if client.refreshedPlayerID != 7 {
		t.Fatalf("refreshed player id = %d, want 7", client.refreshedPlayerID)
	}
	if client.refreshedServerName != "logic-1" {
		t.Fatalf("refreshed server name = %q, want logic-1", client.refreshedServerName)
	}
	if !client.refreshedAt.Equal(updatedAt) {
		t.Fatalf("refreshed at = %v, want %v", client.refreshedAt, updatedAt)
	}
	if client.refreshedTTL != time.Minute {
		t.Fatalf("refreshed ttl = %v, want %v", client.refreshedTTL, time.Minute)
	}
}

func TestStateRepositoryMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: statecontract.ErrPresenceNotFound, want: ErrNotFound},
		{name: "invalid presence", err: statecontract.ErrInvalidPresence, want: ErrInvalidPresence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewStateRepository(&fakeStateClient{err: tt.err})

			_, err := repo.GetPresence(context.Background(), 7)
			if !errors.Is(err, tt.want) {
				t.Fatalf("GetPresence error = %v, want %v", err, tt.want)
			}
		})
	}
}

type fakeStateClient struct {
	presence            *statecontract.Presence
	setPresence         *statecontract.Presence
	setTTL              time.Duration
	clearedPlayerID     int64
	clearedServerName   string
	refreshedPlayerID   int64
	refreshedServerName string
	refreshedAt         time.Time
	refreshedTTL        time.Duration
	err                 error
}

func (f *fakeStateClient) SetPresence(_ context.Context, presence *statecontract.Presence, ttl time.Duration) error {
	f.setPresence = presence
	f.setTTL = ttl
	return f.err
}

func (f *fakeStateClient) GetPresence(_ context.Context, _ int64) (*statecontract.Presence, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.presence, nil
}

func (f *fakeStateClient) ClearPresence(_ context.Context, playerID int64, serverName string) error {
	f.clearedPlayerID = playerID
	f.clearedServerName = serverName
	return f.err
}

func (f *fakeStateClient) RefreshPresence(_ context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	f.refreshedPlayerID = playerID
	f.refreshedServerName = serverName
	f.refreshedAt = updatedAt
	f.refreshedTTL = ttl
	return f.err
}

var _ statecontract.PresenceClient = (*fakeStateClient)(nil)
