package realtime

import (
	"context"
	"server/internal/contract/realtimepb"
	"server/internal/contract/state"
	"server/internal/platform/logging"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const proactivePushRequestID = 0

func toProtoEnvelope(event state.RealtimeEvent) (*realtimepb.ServerEnvelope, bool) {
	switch event.Type {
	case state.RealtimeEventConnectionReplaced:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_ConnectionReplaced{
				ConnectionReplaced: &realtimepb.ConnectionReplaced{},
			},
		}, true
	case state.RealtimeEventFriendPresenceChanged:
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
	case state.RealtimeEventFriendRemoved:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendRemoved{
				FriendRemoved: &realtimepb.FriendRemoved{
					PlayerId: event.ActorPlayerID,
				},
			},
		}, true
	case state.RealtimeEventFriendRequestReceived:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendRequestReceived{
				FriendRequestReceived: &realtimepb.FriendRequestReceived{
					PlayerId: event.ActorPlayerID,
					Nickname: event.ActorPlayerNickname,
				},
			},
		}, true
	case state.RealtimeEventFriendRequestHandled:
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_FriendRequestHandled{
				FriendRequestHandled: &realtimepb.FriendRequestHandled{
					PlayerId: event.ActorPlayerID,
				},
			},
		}, true
	case state.RealtimeEventMatchResult:
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
	case state.RealtimeEventChatMessage:
		if event.ChatMessage == nil {
			return nil, false
		}
		return &realtimepb.ServerEnvelope{
			RequestId: proactivePushRequestID,
			Payload: &realtimepb.ServerEnvelope_ChatMessagePushed{
				ChatMessagePushed: &realtimepb.ChatMessagePushed{
					Message: toProtoStateChatMessage(event.ChatMessage),
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

// Push 将实时事件转换为协议消息并投递到本机玩家连接。
func (l *localRealtimePusher) Push(ctx context.Context, event state.RealtimeEvent) bool {
	if ctx.Err() != nil {
		return false
	}
	serverEnvelope, ok := toProtoEnvelope(event)
	if !ok {
		logging.Warn("realtime event conversion skipped type=%s", event.Type)
		return false
	}
	if event.Type == state.RealtimeEventConnectionReplaced {
		return l.connections.Close(event.TargetPlayerID, serverEnvelope)
	}

	ok = l.connections.Send(event.TargetPlayerID, serverEnvelope)
	if !ok {
		logging.Warn("realtime push target unavailable player_id=%d event=%s", event.TargetPlayerID, event.Type)
	}
	return ok
}

func toProtoStateChatMessage(message *state.ChatMessage) *realtimepb.ChatMessage {
	if message == nil {
		return nil
	}
	return &realtimepb.ChatMessage{
		MessageKey:       message.MessageKey,
		ChannelType:      string(message.ChannelType),
		ChannelKey:       message.ChannelKey,
		SenderId:         message.SenderID,
		ReceiverId:       message.ReceiverID,
		SenderNickname:   message.SenderNickname,
		Content:          message.Content,
		CreatedAt:        toProtoTime(message.CreatedAt),
		ExpiresAt:        toProtoTime(message.ExpiresAt),
		ClientMessageKey: message.ClientMessageKey,
	}
}

func toProtoTime(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
