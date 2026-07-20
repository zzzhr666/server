package httpapi

import (
	"context"
	"server/internal/logic/realtime"
)

type realtimeSubscriber struct {
	serverName string
	bus        realtime.EventBus
	pusher     *localRealtimePusher
}

func newRealtimeSubscriber(serverName string, bus realtime.EventBus, pusher *localRealtimePusher) *realtimeSubscriber {
	return &realtimeSubscriber{
		serverName: serverName,
		bus:        bus,
		pusher:     pusher,
	}
}

func (r *realtimeSubscriber) Run(ctx context.Context) error {
	events, err := r.bus.Subscribe(ctx, r.serverName)
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
			r.pusher.Push(ctx, event)

		}
	}
}
