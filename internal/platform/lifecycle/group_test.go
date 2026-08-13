package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGroupReportsTaskError(t *testing.T) {
	group := NewGroup(context.Background(), 1)
	wantErr := errors.New("serve failed")
	group.Go("test server", func(context.Context) error {
		return wantErr
	})

	err := group.WaitForStop()
	if !errors.Is(err, wantErr) {
		t.Fatalf("WaitForStop() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "test server") {
		t.Errorf("WaitForStop() error = %q, want task name", err)
	}
}

func TestGroupReportsUnexpectedSuccessfulReturn(t *testing.T) {
	group := NewGroup(context.Background(), 1)
	group.Go("test server", func(context.Context) error {
		return nil
	})

	err := group.WaitForStop()
	if !strings.Contains(err.Error(), "test server stopped unexpectedly") {
		t.Errorf("WaitForStop() error = %q, want unexpected stop error", err)
	}
}

func TestGroupCancellationStopsTask(t *testing.T) {
	group := NewGroup(context.Background(), 1)
	group.Go("test server", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	group.Stop()
	if err := group.WaitForStop(); err != nil {
		t.Fatalf("WaitForStop() error = %v, want nil", err)
	}
	if err := group.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestGroupWaitHonorsDeadline(t *testing.T) {
	group := NewGroup(context.Background(), 1)
	block := make(chan struct{})
	group.Go("blocked task", func(context.Context) error {
		<-block
		return nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := group.Wait(ctx)
	close(block)
	group.Stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
