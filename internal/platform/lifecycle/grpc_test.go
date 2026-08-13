package lifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestGracefulStopGRPC(t *testing.T) {
	server := newFakeGRPCServer(false)

	if err := GracefulStopGRPC(context.Background(), server); err != nil {
		t.Fatalf("GracefulStopGRPC() error = %v", err)
	}
	if server.stopCalled() {
		t.Error("GracefulStopGRPC() called Stop after graceful shutdown")
	}
}

func TestGracefulStopGRPCForcesStopAfterCancellation(t *testing.T) {
	server := newFakeGRPCServer(true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := GracefulStopGRPC(ctx, server)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GracefulStopGRPC() error = %v, want %v", err, context.Canceled)
	}
	if !server.stopCalled() {
		t.Error("GracefulStopGRPC() did not call Stop after cancellation")
	}
}

type fakeGRPCServer struct {
	blockGraceful bool
	stopped       chan struct{}
	stopOnce      sync.Once
	mu            sync.Mutex
	forced        bool
}

func newFakeGRPCServer(blockGraceful bool) *fakeGRPCServer {
	return &fakeGRPCServer{
		blockGraceful: blockGraceful,
		stopped:       make(chan struct{}),
	}
}

func (s *fakeGRPCServer) GracefulStop() {
	if s.blockGraceful {
		<-s.stopped
	}
}

func (s *fakeGRPCServer) Stop() {
	s.mu.Lock()
	s.forced = true
	s.mu.Unlock()
	s.stopOnce.Do(func() {
		close(s.stopped)
	})
}

func (s *fakeGRPCServer) stopCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.forced
}
