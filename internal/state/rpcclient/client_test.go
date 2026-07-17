package rpcclient

import (
	"context"
	"net"
	"net/rpc"
	statecontract "server/internal/contract/state"
	"server/internal/state/rpcserver"
	"testing"
)

func TestClientGetAccount(t *testing.T) {
	client := newTestClient(t, &fakeStateClient{
		account: &statecontract.Account{
			Username:     "alice",
			PasswordHash: "hash",
			PlayerID:     7,
		},
	})

	account, err := client.GetAccount(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if account.Username != "alice" {
		t.Fatalf("username = %q, want alice", account.Username)
	}
	if account.PlayerID != 7 {
		t.Fatalf("player id = %d, want 7", account.PlayerID)
	}
}

func newTestClient(t *testing.T, state statecontract.Client) *Client {
	t.Helper()

	server := rpc.NewServer()
	if err := server.RegisterName(rpcserver.ServiceName, rpcserver.NewServer(state)); err != nil {
		t.Fatalf("RegisterName returned error: %v", err)
	}
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})
	go server.ServeConn(serverConn)

	return NewClient(rpc.NewClient(clientConn))
}

type fakeStateClient struct {
	account *statecontract.Account
}

func (f *fakeStateClient) CreateAccount(_ context.Context, _ *statecontract.Account) error {
	return nil
}

func (f *fakeStateClient) GetAccount(_ context.Context, _ string) (*statecontract.Account, error) {
	return f.account, nil
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

func (f *fakeStateClient) CreatePlayer(_ context.Context, _ *statecontract.Player) error {
	return nil
}

func (f *fakeStateClient) GetPlayer(_ context.Context, _ int64) (*statecontract.Player, error) {
	return nil, nil
}

func (f *fakeStateClient) NextPlayerID(_ context.Context) (int64, error) {
	return 0, nil
}

var _ statecontract.Client = (*fakeStateClient)(nil)
