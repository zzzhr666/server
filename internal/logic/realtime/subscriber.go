package realtime

import (
	"context"
	"server/internal/contract/state"
)

type subscriber struct {
	serverName string
	client     state.RealtimeClient
	pusher     *localRealtimePusher
}

func newSubscriber(serverName string, client state.RealtimeClient, manager *connectionManager) *subscriber {
	return &subscriber{
		serverName: serverName,
		client:     client,
		pusher:     newLocalRealtimePusher(manager),
	}
}

func (s *subscriber) Run(ctx context.Context) error {
	serverRoute := state.RealtimeRoute{Type: state.RealtimeRouteServer, ServerName: s.serverName}
	broadcastRoute := state.RealtimeRoute{Type: state.RealtimeRouteBroadcast}
	serverDeliveries, err := s.client.SubscribeRealtime(ctx, serverRoute)
	if err != nil {
		return err
	}
	broadcastDeliveries, err := s.client.SubscribeRealtime(ctx, broadcastRoute)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-serverDeliveries:
			if !ok {
				return nil
			}
			if delivery == nil || delivery.Event == nil || delivery.Route != serverRoute {
				continue
			}
			s.pusher.Push(ctx, *delivery.Event)
		case delivery, ok := <-broadcastDeliveries:
			if !ok {
				return nil
			}
			if delivery == nil || delivery.Event == nil || delivery.Route != broadcastRoute {
				continue
			}
			envelope, ok := toProtoEnvelope(*delivery.Event)
			if !ok {
				continue
			}
			s.pusher.connections.Broadcast(envelope, delivery.Event.ActorPlayerID)
		}
	}
}
