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

// Push 将实时事件写入目标玩家的本地 WebSocket 连接。
func (p *localRealtimePusher) Push(ctx context.Context, event statecontract.RealtimeEvent) bool {
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	msg := toWebSocketEvent(event)
	if event.Type == statecontract.RealtimeEventConnectionReplaced {
		return p.connections.Close(writeCtx, event.TargetPlayerID, msg, websocket.StatusPolicyViolation, "connection replaced")
	}
	return p.connections.SendJSON(writeCtx, event.TargetPlayerID, msg)
}

// toWebSocketEvent 将状态层实时事件转换为网络消息结构。
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
			BattleUDPAddr:  event.BattleUDPAddr,
		}

	default:
		return websocketMessage{Type: event.Type}
	}
}
