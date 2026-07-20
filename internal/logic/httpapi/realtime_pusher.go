package httpapi

import (
	"context"
	"server/internal/logic/realtime"
	"time"
)

type localRealtimePusher struct {
	connections *connManager
}

func newLocalRealtimePusher(connections *connManager) *localRealtimePusher {
	return &localRealtimePusher{
		connections: connections,
	}
}

func (p *localRealtimePusher) Push(ctx context.Context, event realtime.Event) bool {
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.connections.SendJSON(writeCtx, event.TargetPlayerID, toWebSocketEvent(event))
}

func toWebSocketEvent(event realtime.Event) any {
	switch event.Type {
	case realtime.EventFriendPresenceChanged:
		return friendPresenceChangedMessage{
			Type:     event.Type,
			PlayerID: event.ActorPlayerID,
			Online:   event.Online,
			Status:   event.Status,
		}
	case realtime.EventFriendRemoved:
		return friendRemovedMessage{
			Type:     event.Type,
			PlayerID: event.ActorPlayerID,
		}
	default:
		return websocketMessage{Type: event.Type}
	}
}
