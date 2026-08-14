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
	subscriber := newSubscriber("logic-test", &fakeSubscriberClient{subscribeErr: wantErr}, newConnectionManager(), nil)
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
	subscriber := newSubscriber("logic-test", client, connections, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- subscriber.Run(ctx)
	}()
	client.waitForSubscriptions(t, 2)
	client.serverDeliveries <- &statecontract.RealtimeDelivery{
		Route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteServer, ServerName: "logic-test"},
		Event: &statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventFriendRemoved,
			TargetPlayerID: 8,
			ActorPlayerID:  7,
		},
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
	subscriber := newSubscriber("logic-test", client, newConnectionManager(), nil)
	done := make(chan error, 1)
	go func() {
		done <- subscriber.Run(context.Background())
	}()
	client.waitForSubscription(t)
	close(client.serverDeliveries)
	if err := waitForSubscriber(t, done); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func TestSubscriberRunBroadcastsToLocalConnectionsExceptActor(t *testing.T) {
	client := newFakeSubscriberClient()
	connections := newConnectionManager()
	actorServer, actorClient := net.Pipe()
	receiverServer, receiverClient := net.Pipe()
	defer actorServer.Close()
	defer actorClient.Close()
	defer receiverServer.Close()
	defer receiverClient.Close()
	connections.Add(7, newSession(actorServer))
	connections.Add(8, newSession(receiverServer))
	subscriber := newSubscriber("logic-test", client, connections, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- subscriber.Run(ctx)
	}()
	client.waitForSubscriptions(t, 2)
	client.broadcastDeliveries <- &statecontract.RealtimeDelivery{
		Route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteBroadcast},
		Event: &statecontract.RealtimeEvent{
			Type:          statecontract.RealtimeEventChatMessage,
			ActorPlayerID: 7,
			ChatMessage: &statecontract.ChatMessage{
				MessageKey:  "message-1",
				ChannelType: statecontract.ChatChannelWorld,
				ChannelKey:  statecontract.WorldChatChannelKey,
				SenderID:    7,
				Content:     "hello world",
			},
		},
	}

	envelope := readServerEnvelopeOrFail(t, receiverClient)
	message := envelope.GetChatMessagePushed().GetMessage()
	if envelope.GetRequestId() != proactivePushRequestID || message.GetMessageKey() != "message-1" || message.GetContent() != "hello world" {
		t.Fatalf("broadcast envelope = %v, want world chat message", envelope)
	}
	if err := actorClient.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("set actor read deadline: %v", err)
	}
	if _, err := readServerEnvelope(actorClient); err == nil {
		t.Fatal("actor received its own broadcast, want excluded")
	}

	cancel()
	if err := waitForSubscriber(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
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
	mu                  sync.Mutex
	subscribedRoutes    []statecontract.RealtimeRoute
	subscribeErr        error
	serverDeliveries    chan *statecontract.RealtimeDelivery
	broadcastDeliveries chan *statecontract.RealtimeDelivery
	subscribed          chan struct{}
	subscribeOnce       sync.Once
}

func newFakeSubscriberClient() *fakeSubscriberClient {
	return &fakeSubscriberClient{
		serverDeliveries:    make(chan *statecontract.RealtimeDelivery),
		broadcastDeliveries: make(chan *statecontract.RealtimeDelivery),
		subscribed:          make(chan struct{}),
	}
}

func (f *fakeSubscriberClient) PublishRealtime(context.Context, *statecontract.RealtimeDelivery) error {
	return errors.New("unexpected PublishRealtime call")
}

func (f *fakeSubscriberClient) SubscribeRealtime(_ context.Context, route statecontract.RealtimeRoute) (<-chan *statecontract.RealtimeDelivery, error) {
	f.mu.Lock()
	f.subscribedRoutes = append(f.subscribedRoutes, route)
	f.mu.Unlock()
	if f.subscribed != nil {
		f.subscribeOnce.Do(func() { close(f.subscribed) })
	}
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	if route.Type == statecontract.RealtimeRouteBroadcast {
		return f.broadcastDeliveries, nil
	}
	return f.serverDeliveries, nil
}

func (f *fakeSubscriberClient) serverName() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, route := range f.subscribedRoutes {
		if route.Type == statecontract.RealtimeRouteServer {
			return route.ServerName
		}
	}
	return ""
}

func (f *fakeSubscriberClient) waitForSubscription(t *testing.T) {
	f.waitForSubscriptions(t, 1)
}

func (f *fakeSubscriberClient) waitForSubscriptions(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		got := len(f.subscribedRoutes)
		f.mu.Unlock()
		if got >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("SubscribeRealtime calls < %d", want)
}

var _ statecontract.RealtimeClient = (*fakeSubscriberClient)(nil)
