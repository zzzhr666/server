package httpapi

import (
	"context"
	statecontract "server/internal/contract/state"
	"time"

	"github.com/coder/websocket"
)

type localRealtimePusher struct {
	connections *connManager
}

func newLocalRealtimePusher(connections *connManager) *localRealtimePusher {
	return &localRealtimePusher{
		connections: connections,
	}
}

// Push writes a realtime event to the target player's local WebSocket connection.
func (p *localRealtimePusher) Push(ctx context.Context, event statecontract.RealtimeEvent) bool {
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	msg := toWebSocketEvent(event)
	if event.Type == statecontract.RealtimeEventConnectionReplaced {
		return p.connections.Close(writeCtx, event.TargetPlayerID, msg, websocket.StatusPolicyViolation, "connection replaced")
	}
	return p.connections.SendJSON(writeCtx, event.TargetPlayerID, msg)
}

// toWebSocketEvent converts a state realtime event to the wire message shape.
func toWebSocketEvent(event statecontract.RealtimeEvent) any {
	switch event.Type {
	case statecontract.RealtimeEventFriendPresenceChanged:
		return friendPresenceChangedMessage{
			Type:     event.Type,
			PlayerID: event.ActorPlayerID,
			Online:   event.Online,
			Status:   event.Status,
		}
	case statecontract.RealtimeEventFriendRemoved:
		return friendRemovedMessage{
			Type:     event.Type,
			PlayerID: event.ActorPlayerID,
		}
	case statecontract.RealtimeEventFriendRequestReceived:
		return friendRequestReceivedMessage{
			Type:     event.Type,
			PlayerID: event.ActorPlayerID,
		}
	case statecontract.RealtimeEventFriendRequestHandled:
		return friendRequestHandledMessage{
			Type:     event.Type,
			PlayerID: event.ActorPlayerID,
		}
	case statecontract.RealtimeEventConnectionReplaced:
		return connectionReplacedMessage{
			Type: event.Type,
		}
	case statecontract.RealtimeEventMatchResult:
		return matchResultMessage{
			Type:           event.Type,
			Status:         event.MatchStatus,
			RoomName:       event.RoomName,
			Token:          event.MatchToken,
			BattleNodeName: event.BattleNodeName,
			BattleKCPAddr:  event.BattleKCPAddr,
		}

	default:
		return websocketMessage{Type: event.Type}
	}
}
