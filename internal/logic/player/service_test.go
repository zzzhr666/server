package player

import (
	"context"
	"errors"
	"testing"
)

func TestGamePlayerServiceUpdateAvatar(t *testing.T) {
	tests := []struct {
		name       string
		avatar     string
		wantAvatar string
		wantErr    error
	}{
		{
			name:       "normalizes supported avatar",
			avatar:     " mage ",
			wantAvatar: string(AvatarMage),
		},
		{
			name:       "empty uses default avatar",
			avatar:     "",
			wantAvatar: string(DefaultAvatar),
		},
		{
			name:    "rejects unsupported avatar",
			avatar:  "https://example.com/avatar.png",
			wantErr: ErrInvalidAvatar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakePlayerRepository{
				updateResult: &Player{ID: 7, Nickname: "Alice"},
			}
			service := NewService(repo)

			got, err := service.UpdateAvatar(context.Background(), 7, tt.avatar)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateAvatar error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if repo.updateCalls != 0 {
					t.Fatalf("repository update calls = %d, want 0", repo.updateCalls)
				}
				return
			}
			if repo.updateCalls != 1 {
				t.Fatalf("repository update calls = %d, want 1", repo.updateCalls)
			}
			if repo.updateID != 7 {
				t.Fatalf("repository player id = %d, want 7", repo.updateID)
			}
			if repo.updateAvatar != tt.wantAvatar {
				t.Fatalf("repository avatar = %q, want %q", repo.updateAvatar, tt.wantAvatar)
			}
			if got != repo.updateResult {
				t.Fatalf("UpdateAvatar result = %#v, want repository result %#v", got, repo.updateResult)
			}
		})
	}
}

func TestGamePlayerServiceUpdateAvatarCanceledContext(t *testing.T) {
	repo := &fakePlayerRepository{}
	service := NewService(repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.UpdateAvatar(ctx, 7, string(AvatarMage))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("UpdateAvatar error = %v, want %v", err, context.Canceled)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("repository update calls = %d, want 0", repo.updateCalls)
	}
}

func TestGamePlayerServiceUpdateAvatarPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("update failed")
	repo := &fakePlayerRepository{updateErr: wantErr}
	service := NewService(repo)

	_, err := service.UpdateAvatar(context.Background(), 7, string(AvatarWarrior))
	if !errors.Is(err, wantErr) {
		t.Fatalf("UpdateAvatar error = %v, want %v", err, wantErr)
	}
}

type fakePlayerRepository struct {
	updateCalls  int
	updateID     int64
	updateAvatar string
	updateResult *Player
	updateErr    error
}

func (f *fakePlayerRepository) NextID(context.Context) (int64, error) {
	return 0, errors.New("unexpected NextID call")
}

func (f *fakePlayerRepository) Create(context.Context, *Player) error {
	return errors.New("unexpected Create call")
}

func (f *fakePlayerRepository) Get(context.Context, int64) (*Player, error) {
	return nil, errors.New("unexpected Get call")
}

func (f *fakePlayerRepository) UpdateAvatar(_ context.Context, id int64, avatar string) (*Player, error) {
	f.updateCalls++
	f.updateID = id
	f.updateAvatar = avatar
	return f.updateResult, f.updateErr
}

var _ Repository = (*fakePlayerRepository)(nil)
