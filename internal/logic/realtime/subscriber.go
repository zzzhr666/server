package realtime

import (
	"context"
	statecontract "server/internal/contract/state"
)

type subscriber struct {
	serverName string
	client     statecontract.RealtimeClient
	pusher     *localRealtimePusher
}

func newSubscriber(serverName string, client statecontract.RealtimeClient, manager *connectionManager) *subscriber {
	return &subscriber{
		serverName: serverName,
		client:     client,
		pusher:     newLocalRealtimePusher(manager),
	}
}

func (s *subscriber) Run(ctx context.Context) error {
	events, err := s.client.SubscribeRealtime(ctx, s.serverName)
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
			s.pusher.Push(ctx, *event)
		}
	}
}
