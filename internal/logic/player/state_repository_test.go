package player

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
)

func TestStateRepositoryNextID(t *testing.T) {
	client := newFakeStateClient()
	client.nextPlayerID = 7
	repo := NewStateRepository(client)

	id, err := repo.NextID(context.Background())
	if err != nil {
		t.Fatalf("NextID returned error: %v", err)
	}
	if id != 7 {
		t.Fatalf("id = %d, want 7", id)
	}
}

func TestStateRepositoryCreateAndGet(t *testing.T) {
	client := newFakeStateClient()
	repo := NewStateRepository(client)
	player := &Player{
		ID:       7,
		Nickname: "Alice",
		Avatar:   "alice.png",
		Email:    "alice@example.com",
		Phone:    "13800000000",
	}

	if err := repo.Create(context.Background(), player); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	got, err := repo.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != player.ID {
		t.Fatalf("id = %d, want %d", got.ID, player.ID)
	}
	if got.Nickname != player.Nickname {
		t.Fatalf("nickname = %q, want %q", got.Nickname, player.Nickname)
	}
	if got.Avatar != player.Avatar {
		t.Fatalf("avatar = %q, want %q", got.Avatar, player.Avatar)
	}
	if got.Email != player.Email {
		t.Fatalf("email = %q, want %q", got.Email, player.Email)
	}
	if got.Phone != player.Phone {
		t.Fatalf("phone = %q, want %q", got.Phone, player.Phone)
	}
}

func TestStateRepositoryGetMissingPlayer(t *testing.T) {
	repo := NewStateRepository(newFakeStateClient())

	_, err := repo.Get(context.Background(), 7)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get error = %v, want %v", err, ErrNotFound)
	}
}

func TestStateRepositoryUpdateAvatar(t *testing.T) {
	client := newFakeStateClient()
	client.players[7] = &statecontract.Player{
		ID:       7,
		Nickname: "Alice",
		Avatar:   string(AvatarAdventurer),
		Email:    "alice@example.com",
		Phone:    "13800000000",
		Coins:    100,
	}
	repo := NewStateRepository(client)

	got, err := repo.UpdateAvatar(context.Background(), 7, string(AvatarMage))
	if err != nil {
		t.Fatalf("UpdateAvatar returned error: %v", err)
	}
	if got.ID != 7 || got.Nickname != "Alice" || got.Avatar != string(AvatarMage) ||
		got.Email != "alice@example.com" || got.Phone != "13800000000" || got.Coins != 100 {
		t.Fatalf("updated player = %#v, want complete profile with mage avatar", got)
	}
}

func TestStateRepositoryUpdateAvatarMissingPlayer(t *testing.T) {
	repo := NewStateRepository(newFakeStateClient())

	_, err := repo.UpdateAvatar(context.Background(), 7, string(AvatarMage))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAvatar error = %v, want %v", err, ErrNotFound)
	}
}

type fakeStateClient struct {
	nextPlayerID int64
	players      map[int64]*statecontract.Player
}

func (f *fakeStateClient) UpdatePlayerAvatar(_ context.Context, playerID int64, avatar string) (*statecontract.Player, error) {
	player, ok := f.players[playerID]
	if !ok {
		return nil, statecontract.ErrPlayerNotFound
	}
	updated := new(*player)
	updated.Avatar = avatar
	f.players[playerID] = updated
	return new(*updated), nil
}

func newFakeStateClient() *fakeStateClient {
	return &fakeStateClient{
		players: make(map[int64]*statecontract.Player),
	}
}

func (f *fakeStateClient) CreateAccount(_ context.Context, _ *statecontract.Account) error {
	return nil
}

func (f *fakeStateClient) GetAccount(_ context.Context, _ string) (*statecontract.Account, error) {
	return nil, nil
}

func (f *fakeStateClient) RegisterAccount(_ context.Context, _ statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	return nil, nil
}

func (f *fakeStateClient) CreateSession(_ context.Context, _ *statecontract.Session) error {
	return nil
}

func (f *fakeStateClient) GetSession(_ context.Context, _ string) (*statecontract.Session, error) {
	return nil, nil
}

func (f *fakeStateClient) DeleteSession(_ context.Context, _ string) error {
	return nil
}

func (f *fakeStateClient) CreatePlayer(_ context.Context, player *statecontract.Player) error {
	f.players[player.ID] = new(*player)
	return nil
}

func (f *fakeStateClient) GetPlayer(_ context.Context, id int64) (*statecontract.Player, error) {
	player, ok := f.players[id]
	if !ok {
		return nil, statecontract.ErrPlayerNotFound
	}
	return new(*player), nil
}

func (f *fakeStateClient) NextPlayerID(_ context.Context) (int64, error) {
	return f.nextPlayerID, nil
}

var _ statecontract.Client = (*fakeStateClient)(nil)
