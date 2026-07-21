package httpapi

import (
	"context"
	statecontract "server/internal/contract/state"
)

type realtimeSubscriber struct {
	serverName string
	client     statecontract.RealtimeClient
	pusher     *localRealtimePusher
}

func newRealtimeSubscriber(serverName string, client statecontract.RealtimeClient, pusher *localRealtimePusher) *realtimeSubscriber {
	return &realtimeSubscriber{
		serverName: serverName,
		client:     client,
		pusher:     pusher,
	}
}

// Run subscribes to this logic-server's realtime channel and forwards events locally.
func (r *realtimeSubscriber) Run(ctx context.Context) error {
	events, err := r.client.SubscribeRealtime(ctx, r.serverName)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if event == nil {
				continue
			}
			r.pusher.Push(ctx, *event)

		}
	}
}
