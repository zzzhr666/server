package redisstore

import (
	"context"
	"errors"
	"net"
	"os/exec"
	statecontract "server/internal/contract/state"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestFriendRequestLifecycleAccept(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)

	if err := store.SendFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}

	requestValues, err := client.HGetAll(ctx, friendRequestKey(7, 8)).Result()
	if err != nil {
		t.Fatalf("HGetAll friend request returned error: %v", err)
	}
	if requestValues["from_player_id"] != "7" {
		t.Fatalf("from_player_id = %q, want 7", requestValues["from_player_id"])
	}
	if requestValues["to_player_id"] != "8" {
		t.Fatalf("to_player_id = %q, want 8", requestValues["to_player_id"])
	}
	if requestValues["created_at"] == "" {
		t.Fatalf("created_at is empty")
	}

	assertZSetMembers(t, ctx, client, friendIncomingKey(8), []string{"7"})
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), []string{"8"})

	incoming, err := store.ListIncomingFriendRequests(ctx, 8)
	if err != nil {
		t.Fatalf("ListIncomingFriendRequests returned error: %v", err)
	}
	assertFriendRequests(t, incoming, []friendRequestWant{{from: 7, to: 8}})

	outgoing, err := store.ListOutgoingFriendRequests(ctx, 7)
	if err != nil {
		t.Fatalf("ListOutgoingFriendRequests returned error: %v", err)
	}
	assertFriendRequests(t, outgoing, []friendRequestWant{{from: 7, to: 8}})

	if err := store.SendFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendRequestExists) {
		t.Fatalf("duplicate SendFriendRequest error = %v, want %v", err, statecontract.ErrFriendRequestExists)
	}
	if err := store.SendFriendRequest(ctx, 8, 7); !errors.Is(err, statecontract.ErrFriendRequestExists) {
		t.Fatalf("reverse SendFriendRequest error = %v, want %v", err, statecontract.ErrFriendRequestExists)
	}

	if err := store.AcceptFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("AcceptFriendRequest returned error: %v", err)
	}

	assertSetMembers(t, ctx, client, friendsKey(7), []string{"8"})
	assertSetMembers(t, ctx, client, friendsKey(8), []string{"7"})
	assertHashMissing(t, ctx, client, friendRequestKey(7, 8))
	assertZSetMembers(t, ctx, client, friendIncomingKey(8), nil)
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), nil)

	friendIDs, err := store.ListFriendIDs(ctx, 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	assertInt64Set(t, friendIDs, []int64{8})

	if err := store.SendFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendAlreadyExists) {
		t.Fatalf("SendFriendRequest for existing friends error = %v, want %v", err, statecontract.ErrFriendAlreadyExists)
	}
}

func TestFriendRequestLifecycleReject(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)

	if err := store.SendFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}
	if err := store.RejectFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("RejectFriendRequest returned error: %v", err)
	}

	assertHashMissing(t, ctx, client, friendRequestKey(7, 8))
	assertZSetMembers(t, ctx, client, friendIncomingKey(8), nil)
	assertZSetMembers(t, ctx, client, friendOutgoingKey(7), nil)
	assertSetMembers(t, ctx, client, friendsKey(7), nil)
	assertSetMembers(t, ctx, client, friendsKey(8), nil)

	if err := store.RejectFriendRequest(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendRequestNotFound) {
		t.Fatalf("second RejectFriendRequest error = %v, want %v", err, statecontract.ErrFriendRequestNotFound)
	}
}

func TestDeleteFriendRemovesBothDirections(t *testing.T) {
	ctx := context.Background()
	store, client := newRedisTestStore(t)

	if err := store.SendFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("SendFriendRequest returned error: %v", err)
	}
	if err := store.AcceptFriendRequest(ctx, 7, 8); err != nil {
		t.Fatalf("AcceptFriendRequest returned error: %v", err)
	}

	if err := store.DeleteFriend(ctx, 7, 8); err != nil {
		t.Fatalf("DeleteFriend returned error: %v", err)
	}

	assertSetMembers(t, ctx, client, friendsKey(7), nil)
	assertSetMembers(t, ctx, client, friendsKey(8), nil)

	friendIDs, err := store.ListFriendIDs(ctx, 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	assertInt64Set(t, friendIDs, nil)

	if err := store.DeleteFriend(ctx, 7, 8); !errors.Is(err, statecontract.ErrFriendNotFound) {
		t.Fatalf("second DeleteFriend error = %v, want %v", err, statecontract.ErrFriendNotFound)
	}
}

func TestFriendMethodsValidateInput(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisTestStore(t)

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "send self",
			run: func() error {
				return store.SendFriendRequest(ctx, 7, 7)
			},
		},
		{
			name: "list incoming zero player",
			run: func() error {
				_, err := store.ListIncomingFriendRequests(ctx, 0)
				return err
			},
		},
		{
			name: "list outgoing zero player",
			run: func() error {
				_, err := store.ListOutgoingFriendRequests(ctx, 0)
				return err
			},
		},
		{
			name: "accept self",
			run: func() error {
				return store.AcceptFriendRequest(ctx, 7, 7)
			},
		},
		{
			name: "reject self",
			run: func() error {
				return store.RejectFriendRequest(ctx, 7, 7)
			},
		},
		{
			name: "list friend ids zero player",
			run: func() error {
				_, err := store.ListFriendIDs(ctx, 0)
				return err
			},
		},
		{
			name: "delete self",
			run: func() error {
				return store.DeleteFriend(ctx, 7, 7)
			},
		},
		{
			name: "delete zero friend",
			run: func() error {
				return store.DeleteFriend(ctx, 7, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, statecontract.ErrInvalidFriendRequest) {
				t.Fatalf("error = %v, want %v", err, statecontract.ErrInvalidFriendRequest)
			}
		})
	}
}

type friendRequestWant struct {
	from int64
	to   int64
}

func newRedisTestStore(t *testing.T) (*Store, *redis.Client) {
	t.Helper()

	redisServerPath, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server binary not found")
	}

	addr := freeRedisAddr(t)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split redis address %q: %v", addr, err)
	}

	cmd := exec.Command(
		redisServerPath,
		"--bind", "127.0.0.1",
		"--port", port,
		"--save", "",
		"--appendonly", "no",
		"--dir", t.TempDir(),
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start redis-server: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-done
	})

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("redis-server exited before accepting connections: %v", err)
		case <-ctx.Done():
			t.Fatalf("redis-server did not start: %v", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}

	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("FlushDB returned error: %v", err)
	}
	return NewStore(client), client
}

func freeRedisAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free Redis port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func assertFriendRequests(t *testing.T, got []*statecontract.FriendRequest, want []friendRequestWant) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("friend requests = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].FromPlayerID != want[i].from {
			t.Fatalf("request %d from = %d, want %d", i, got[i].FromPlayerID, want[i].from)
		}
		if got[i].ToPlayerID != want[i].to {
			t.Fatalf("request %d to = %d, want %d", i, got[i].ToPlayerID, want[i].to)
		}
		if got[i].CreatedAt.IsZero() {
			t.Fatalf("request %d created at is zero", i)
		}
	}
}

func assertHashMissing(t *testing.T, ctx context.Context, client *redis.Client, key string) {
	t.Helper()

	values, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		t.Fatalf("HGetAll %s returned error: %v", key, err)
	}
	if len(values) != 0 {
		t.Fatalf("hash %s = %v, want missing", key, values)
	}
}

func assertSetMembers(t *testing.T, ctx context.Context, client *redis.Client, key string, want []string) {
	t.Helper()

	got, err := client.SMembers(ctx, key).Result()
	if err != nil {
		t.Fatalf("SMembers %s returned error: %v", key, err)
	}
	assertStringSet(t, got, want)
}

func assertZSetMembers(t *testing.T, ctx context.Context, client *redis.Client, key string, want []string) {
	t.Helper()

	got, err := client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		t.Fatalf("ZRange %s returned error: %v", key, err)
	}
	assertStringSet(t, got, want)
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("members = %v, want %v", got, want)
	}
	counts := make(map[string]int, len(want))
	for _, value := range want {
		counts[value]++
	}
	for _, value := range got {
		counts[value]--
	}
	for value, count := range counts {
		if count != 0 {
			t.Fatalf("members = %v, want %v; %s count diff %d", got, want, value, count)
		}
	}
}

func assertInt64Set(t *testing.T, got, want []int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	counts := make(map[int64]int, len(want))
	for _, value := range want {
		counts[value]++
	}
	for _, value := range got {
		counts[value]--
	}
	for value, count := range counts {
		if count != 0 {
			t.Fatalf("ids = %v, want %v; %d count diff %d", got, want, value, count)
		}
	}
}
