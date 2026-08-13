package realtime

import (
	"context"
	"server/internal/contract/state"
	"server/internal/platform/logging"
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
		logging.Error("subscribe realtime server route failed server=%s: %v", s.serverName, err)
		return err
	}
	broadcastDeliveries, err := s.client.SubscribeRealtime(ctx, broadcastRoute)
	if err != nil {
		logging.Error("subscribe realtime broadcast route failed server=%s: %v", s.serverName, err)
		return err
	}
	logging.Info("realtime subscriber started server=%s", s.serverName)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-serverDeliveries:
			if !ok {
				logging.Warn("realtime server subscription closed server=%s", s.serverName)
				return nil
			}
			if delivery == nil || delivery.Event == nil || delivery.Route != serverRoute {
				continue
			}
			if !s.pusher.Push(ctx, *delivery.Event) {
				logging.Warn("realtime server delivery push failed server=%s target_player_id=%d", s.serverName, delivery.Event.TargetPlayerID)
			}
		case delivery, ok := <-broadcastDeliveries:
			if !ok {
				logging.Warn("realtime broadcast subscription closed server=%s", s.serverName)
				return nil
			}
			if delivery == nil || delivery.Event == nil || delivery.Route != broadcastRoute {
				continue
			}
			envelope, ok := toProtoEnvelope(*delivery.Event)
			if !ok {
				continue
			}
			count := s.pusher.connections.Broadcast(envelope, delivery.Event.ActorPlayerID)
			logging.Debug("realtime broadcast delivered server=%s recipients=%d", s.serverName, count)
		}
	}
}
