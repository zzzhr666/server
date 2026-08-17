package realtime

import (
	"context"
	"net"
	"sync"
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

func TestServerServeClosesAcceptedSessionWhenContextCanceled(t *testing.T) {
	listener := newTestListener(t)
	presenceService := &fakeHandlerPresence{}
	handler := NewHandler(HandlerConfig{
		AuthService:     &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		PresenceService: presenceService,
		ServerName:      "logic-test",
		IdleTimeout:     time.Minute,
	})
	server := NewServer(listener, handler)
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := serveServer(t, server, ctx)

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial TCP server: %v", err)
	}
	defer client.Close()
	authenticateSession(t, client)

	cancel()
	waitForServer(t, serveDone)

	if _, ok := handler.connManager.Get(7); ok {
		t.Fatal("accepted session remains registered after server shutdown")
	}
	if got := presenceService.offlineCallCount(); got != 1 {
		t.Fatalf("MarkOffline calls = %d, want 1 after server shutdown", got)
	}
	if err := client.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("set client read deadline: %v", err)
	}
	if _, err := readServerEnvelope(client); err == nil {
		t.Fatal("client read error = nil, want closed connection")
	} else if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
		t.Fatalf("client read error = %v, want connection close instead of timeout", err)
	}
}

func TestServerConcurrentCloseClosesUnauthenticatedSessions(t *testing.T) {
	listener := newTestListener(t)
	handler := NewHandler(HandlerConfig{
		AuthService:           &fakeHandlerAuth{},
		AuthenticationTimeout: time.Minute,
	})
	server := NewServer(listener, handler)
	serveDone := serveServer(t, server, context.Background())

	clients := make([]net.Conn, 0, 2)
	for range 2 {
		client, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatalf("dial TCP server: %v", err)
		}
		defer client.Close()
		clients = append(clients, client)
	}
	waitForTrackedSessionCount(t, server, len(clients))

	var closers sync.WaitGroup
	for range 8 {
		closers.Add(1)
		go func() {
			defer closers.Done()
			server.Close()
		}()
	}
	closers.Wait()
	waitForServer(t, serveDone)

	if got := trackedSessionCount(server); got != 0 {
		t.Fatalf("tracked sessions = %d, want 0 after concurrent close", got)
	}
	for _, client := range clients {
		if err := client.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			t.Fatalf("set client read deadline: %v", err)
		}
		if _, err := client.Read(make([]byte, 1)); err == nil {
			t.Fatal("client read error = nil, want closed connection")
		} else if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
			t.Fatalf("client read error = %v, want connection close instead of timeout", err)
		}
	}
}

func trackedSessionCount(server *Server) int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return len(server.sessions)
}

func waitForTrackedSessionCount(t *testing.T, server *Server, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if trackedSessionCount(server) == count {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("tracked sessions = %d, want %d", trackedSessionCount(server), count)
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
