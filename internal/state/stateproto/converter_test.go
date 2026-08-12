package stateproto

import (
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestRealtimeDeliveryRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	want := &statecontract.RealtimeDelivery{
		Route: statecontract.RealtimeRoute{
			Type:       statecontract.RealtimeRouteServer,
			ServerName: "logic-2",
		},
		Event: &statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventChatMessage,
			TargetPlayerID: 8,
			ActorPlayerID:  7,
			ChatMessage: &statecontract.ChatMessage{
				MessageKey:       "message-1",
				ChannelType:      statecontract.ChatChannelDirect,
				ChannelKey:       "direct:7:8",
				SenderID:         7,
				ReceiverID:       8,
				SenderNickname:   "Alice",
				Content:          "hello",
				CreatedAt:        createdAt,
				ExpiresAt:        createdAt.Add(statecontract.DirectChatRetention),
				ClientMessageKey: "client-1",
			},
		},
	}

	got := FromProtoRealtimeDelivery(ToProtoRealtimeDelivery(want))
	if got == nil || got.Event == nil || got.Event.ChatMessage == nil {
		t.Fatalf("delivery round trip = %+v, want complete delivery", got)
	}
	if got.Route != want.Route || got.Event.Type != want.Event.Type || got.Event.TargetPlayerID != want.Event.TargetPlayerID || got.Event.ActorPlayerID != want.Event.ActorPlayerID {
		t.Fatalf("delivery round trip = %+v, want route and event %+v", got, want)
	}
	message := got.Event.ChatMessage
	wantMessage := want.Event.ChatMessage
	if message.MessageKey != wantMessage.MessageKey || message.ChannelType != wantMessage.ChannelType || message.ChannelKey != wantMessage.ChannelKey || message.SenderID != wantMessage.SenderID || message.ReceiverID != wantMessage.ReceiverID || message.SenderNickname != wantMessage.SenderNickname || message.Content != wantMessage.Content || message.ClientMessageKey != wantMessage.ClientMessageKey || !message.CreatedAt.Equal(wantMessage.CreatedAt) || !message.ExpiresAt.Equal(wantMessage.ExpiresAt) {
		t.Fatalf("chat message round trip = %+v, want %+v", message, wantMessage)
	}
}

func TestRealtimeDeliveryConvertersHandleNil(t *testing.T) {
	if ToProtoRealtimeDelivery(nil) != nil {
		t.Fatal("ToProtoRealtimeDelivery(nil) != nil")
	}
	if FromProtoRealtimeDelivery(nil) != nil {
		t.Fatal("FromProtoRealtimeDelivery(nil) != nil")
	}
}
