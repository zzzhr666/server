package realtime

import (
	"context"
	"net"
	"testing"
	"time"

	"server/internal/logic/auth"
)

func TestServerServeHandlesAcceptedSession(t *testing.T) {
	listener := newTestListener(t)
	authService := &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}}
	handler := NewHandler(HandlerConfig{
		AuthService:     authService,
		PresenceService: &fakeHandlerPresence{},
		ServerName:      "logic-test",
	})
	server := NewServer(listener, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := serveServer(t, server, ctx)

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial TCP server: %v", err)
	}
	defer client.Close()
	writeClientEnvelope(t, client, authenticateEnvelope(1, "session-token"))
	response := readServerEnvelopeOrFail(t, client)
	if response.GetRequestId() != 1 || response.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authentication response = %v, want player 7 for request 1", response)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	cancel()
	waitForServer(t, serveDone)
}

func TestServerServeStopsWhenContextCanceled(t *testing.T) {
	listener := newTestListener(t)
	server := NewServer(listener, nil)
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := serveServer(t, server, ctx)

	cancel()
	waitForServer(t, serveDone)
}

func newTestListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen TCP: %v", err)
	}
	return listener
}

func serveServer(t *testing.T, server *Server, ctx context.Context) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- server.Serve(ctx)
	}()
	return done
}

func waitForServer(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not stop")
	}
}
