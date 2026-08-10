package realtime

import (
	"context"
	"errors"
	"net"
	statecontract "server/internal/contract/state"
	"sync"
	"testing"
	"time"
)

func TestHandlerRunRealtimeSubscriberWithoutClient(t *testing.T) {
	handler := NewHandler(HandlerConfig{})
	if err := handler.RunRealtimeSubscriber(context.Background()); err != nil {
		t.Fatalf("RunRealtimeSubscriber() error = %v, want nil", err)
	}
}

func TestSubscriberRunReturnsSubscribeError(t *testing.T) {
	wantErr := errors.New("state unavailable")
	subscriber := newSubscriber("logic-test", &fakeSubscriberClient{subscribeErr: wantErr}, newConnectionManager())
	if err := subscriber.Run(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

func TestSubscriberRunForwardsEventAndStopsOnCancellation(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	connections := newConnectionManager()
	connections.Add(8, newSession(serverConn))
	client := newFakeSubscriberClient()
	subscriber := newSubscriber("logic-test", client, connections)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- subscriber.Run(ctx)
	}()
	client.waitForSubscription(t)
	client.events <- &statecontract.RealtimeEvent{
		Type:           statecontract.RealtimeEventFriendRemoved,
		TargetPlayerID: 8,
		ActorPlayerID:  7,
	}

	envelope := readServerEnvelopeOrFail(t, clientConn)
	if envelope.GetFriendRemoved().GetPlayerId() != 7 {
		t.Fatalf("forwarded envelope = %v, want friend removed for player 7", envelope)
	}
	if client.serverName() != "logic-test" {
		t.Fatalf("SubscribeRealtime server name = %q, want %q", client.serverName(), "logic-test")
	}

	cancel()
	if err := waitForSubscriber(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
	}
}

func TestSubscriberRunReturnsNilWhenEventsClose(t *testing.T) {
	client := newFakeSubscriberClient()
	subscriber := newSubscriber("logic-test", client, newConnectionManager())
	done := make(chan error, 1)
	go func() {
		done <- subscriber.Run(context.Background())
	}()
	client.waitForSubscription(t)
	close(client.events)
	if err := waitForSubscriber(t, done); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func waitForSubscriber(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("subscriber did not stop")
		return nil
	}
}

type fakeSubscriberClient struct {
	mu             sync.Mutex
	subscribedName string
	subscribeErr   error
	events         chan *statecontract.RealtimeEvent
	subscribed     chan struct{}
}

func newFakeSubscriberClient() *fakeSubscriberClient {
	return &fakeSubscriberClient{
		events:     make(chan *statecontract.RealtimeEvent),
		subscribed: make(chan struct{}),
	}
}

func (f *fakeSubscriberClient) PublishRealtimeToServer(context.Context, string, *statecontract.RealtimeEvent) error {
	return errors.New("unexpected PublishRealtimeToServer call")
}

func (f *fakeSubscriberClient) SubscribeRealtime(_ context.Context, serverName string) (<-chan *statecontract.RealtimeEvent, error) {
	f.mu.Lock()
	f.subscribedName = serverName
	f.mu.Unlock()
	if f.subscribed != nil {
		close(f.subscribed)
	}
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	return f.events, nil
}

func (f *fakeSubscriberClient) serverName() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subscribedName
}

func (f *fakeSubscriberClient) waitForSubscription(t *testing.T) {
	t.Helper()
	select {
	case <-f.subscribed:
	case <-time.After(time.Second):
		t.Fatal("SubscribeRealtime was not called")
	}
}

var _ statecontract.RealtimeClient = (*fakeSubscriberClient)(nil)
