package realtime

import (
	"context"
	"server/internal/contract/realtimepb"
	statecontract "server/internal/contract/state"
)

const proactivePushRequestID = 0

func toProtoEnvelope(event statecontract.RealtimeEvent) (*realtimepb.ServerEnvelope, bool) {
	switch event.Type {
	case statecontract.RealtimeEventConnectionReplaced:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_ConnectionReplaced{
				ConnectionReplaced: &realtimepb.ConnectionReplaced{},
			},
		}, true
	case statecontract.RealtimeEventFriendPresenceChanged:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendPresenceChanged{
				FriendPresenceChanged: &realtimepb.FriendPresenceChanged{
					PlayerId: event.ActorPlayerID,
					Online:   event.Online,
					Status:   event.Status,
				},
			},
		}, true
	case statecontract.RealtimeEventFriendRemoved:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendRemoved{
				FriendRemoved: &realtimepb.FriendRemoved{
					PlayerId: event.ActorPlayerID,
				},
			},
		}, true
	case statecontract.RealtimeEventFriendRequestReceived:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendRequestReceived{
				FriendRequestReceived: &realtimepb.FriendRequestReceived{
					PlayerId: event.ActorPlayerID,
				},
			},
		}, true
	case statecontract.RealtimeEventFriendRequestHandled:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendRequestHandled{
				FriendRequestHandled: &realtimepb.FriendRequestHandled{
					PlayerId: event.ActorPlayerID,
				},
			},
		}, true
	case statecontract.RealtimeEventMatchResult:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_MatchResult{
				MatchResult: &realtimepb.MatchResult{
					Status:         event.MatchStatus,
					RoomName:       event.RoomName,
					Token:          event.MatchToken,
					BattleNodeName: event.BattleNodeName,
					BattleUdpAddr:  event.BattleUDPAddr,
				},
			},
		}, true
	}
	return nil, false
}

type localRealtimePusher struct {
	connections *connectionManager
}

func newLocalRealtimePusher(connections *connectionManager) *localRealtimePusher {
	return &localRealtimePusher{
		connections: connections,
	}
}

func (l *localRealtimePusher) Push(ctx context.Context, event statecontract.RealtimeEvent) bool {
	if ctx.Err() != nil {
		return false
	}
	serverEnvelope, ok := toProtoEnvelope(event)
	if !ok {
		return false
	}
	if event.Type == statecontract.RealtimeEventConnectionReplaced {
		return l.connections.Close(event.TargetPlayerID, serverEnvelope)
	}

	return l.connections.Send(event.TargetPlayerID, serverEnvelope)
}
