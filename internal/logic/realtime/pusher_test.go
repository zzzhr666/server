package realtime

import (
	"context"
	"net"
	"server/internal/contract/realtimepb"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestToProtoEnvelopeMapsRealtimeEvents(t *testing.T) {
	tests := []struct {
		name  string
		event statecontract.RealtimeEvent
		check func(*testing.T, *statecontract.RealtimeEvent, *realtimepb.ServerEnvelope)
	}{
		{
			name:  "connection replaced",
			event: statecontract.RealtimeEvent{Type: statecontract.RealtimeEventConnectionReplaced},
			check: func(t *testing.T, _ *statecontract.RealtimeEvent, envelope *realtimepb.ServerEnvelope) {
				if envelope.GetConnectionReplaced() == nil {
					t.Fatal("connection replaced payload = nil")
				}
			},
		},
		{
			name:  "friend presence changed",
			event: statecontract.RealtimeEvent{Type: statecontract.RealtimeEventFriendPresenceChanged, ActorPlayerID: 7, Online: true, Status: "online"},
			check: func(t *testing.T, event *statecontract.RealtimeEvent, envelope *realtimepb.ServerEnvelope) {
				payload := envelope.GetFriendPresenceChanged()
				if payload == nil || payload.GetPlayerId() != event.ActorPlayerID || !payload.GetOnline() || payload.GetStatus() != "online" {
					t.Fatalf("friend presence payload = %v, want actor 7 online", payload)
				}
			},
		},
		{
			name:  "friend removed",
			event: statecontract.RealtimeEvent{Type: statecontract.RealtimeEventFriendRemoved, ActorPlayerID: 7},
			check: func(t *testing.T, event *statecontract.RealtimeEvent, envelope *realtimepb.ServerEnvelope) {
				payload := envelope.GetFriendRemoved()
				if payload == nil || payload.GetPlayerId() != event.ActorPlayerID {
					t.Fatalf("friend removed payload = %v, want actor 7", payload)
				}
			},
		},
		{
			name:  "friend request received",
			event: statecontract.RealtimeEvent{Type: statecontract.RealtimeEventFriendRequestReceived, ActorPlayerID: 7},
			check: func(t *testing.T, event *statecontract.RealtimeEvent, envelope *realtimepb.ServerEnvelope) {
				payload := envelope.GetFriendRequestReceived()
				if payload == nil || payload.GetPlayerId() != event.ActorPlayerID {
					t.Fatalf("friend request received payload = %v, want actor 7", payload)
				}
			},
		},
		{
			name:  "friend request handled",
			event: statecontract.RealtimeEvent{Type: statecontract.RealtimeEventFriendRequestHandled, ActorPlayerID: 7},
			check: func(t *testing.T, event *statecontract.RealtimeEvent, envelope *realtimepb.ServerEnvelope) {
				payload := envelope.GetFriendRequestHandled()
				if payload == nil || payload.GetPlayerId() != event.ActorPlayerID {
					t.Fatalf("friend request handled payload = %v, want actor 7", payload)
				}
			},
		},
		{
			name:  "match result",
			event: statecontract.RealtimeEvent{Type: statecontract.RealtimeEventMatchResult, MatchStatus: "matched", RoomName: "room-7-8", MatchToken: "token", BattleNodeName: "battle-1", BattleUDPAddr: "127.0.0.1:7001"},
			check: func(t *testing.T, _ *statecontract.RealtimeEvent, envelope *realtimepb.ServerEnvelope) {
				payload := envelope.GetMatchResult()
				if payload == nil || payload.GetStatus() != "matched" || payload.GetRoomName() != "room-7-8" || payload.GetToken() != "token" || payload.GetBattleNodeName() != "battle-1" || payload.GetBattleUdpAddr() != "127.0.0.1:7001" {
					t.Fatalf("match result payload = %v, want complete match allocation", payload)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envelope, ok := toProtoEnvelope(test.event)
			if !ok || envelope == nil {
				t.Fatal("toProtoEnvelope() = no envelope, want payload")
			}
			if envelope.GetRequestId() != proactivePushRequestID {
				t.Fatalf("request ID = %d, want 0", envelope.GetRequestId())
			}
			test.check(t, &test.event, envelope)
		})
	}
}

func TestToProtoEnvelopeRejectsUnknownEvent(t *testing.T) {
	envelope, ok := toProtoEnvelope(statecontract.RealtimeEvent{Type: "unknown"})
	if ok || envelope != nil {
		t.Fatalf("toProtoEnvelope() = (%v, %t), want (nil, false)", envelope, ok)
	}
}

func TestLocalRealtimePusherPushesAndClosesReplacedConnection(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	connections := newConnectionManager()
	connections.Add(8, newSession(serverConn))
	pusher := newLocalRealtimePusher(connections)

	done := make(chan bool, 1)
	go func() {
		done <- pusher.Push(context.Background(), statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventConnectionReplaced,
			TargetPlayerID: 8,
		})
	}()

	envelope := readServerEnvelopeOrFail(t, clientConn)
	if envelope.GetRequestId() != proactivePushRequestID || envelope.GetConnectionReplaced() == nil {
		t.Fatalf("pushed envelope = %v, want connection replaced", envelope)
	}
	if !waitForPush(t, done) {
		t.Fatal("Push() = false, want true")
	}
	if _, err := readServerEnvelope(clientConn); err == nil {
		t.Fatal("connection remained open after replacement")
	}
}

func TestLocalRealtimePusherPushesEventToTargetConnection(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	connections := newConnectionManager()
	connections.Add(8, newSession(serverConn))
	pusher := newLocalRealtimePusher(connections)

	done := make(chan bool, 1)
	go func() {
		done <- pusher.Push(context.Background(), statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventFriendRemoved,
			TargetPlayerID: 8,
			ActorPlayerID:  7,
		})
	}()

	envelope := readServerEnvelopeOrFail(t, clientConn)
	if envelope.GetRequestId() != proactivePushRequestID || envelope.GetFriendRemoved().GetPlayerId() != 7 {
		t.Fatalf("pushed envelope = %v, want friend removed for actor 7", envelope)
	}
	if !waitForPush(t, done) {
		t.Fatal("Push() = false, want true")
	}
}

func TestLocalRealtimePusherRejectsCanceledContextAndMissingConnection(t *testing.T) {
	pusher := newLocalRealtimePusher(newConnectionManager())
	event := statecontract.RealtimeEvent{Type: statecontract.RealtimeEventFriendRemoved, TargetPlayerID: 8, ActorPlayerID: 7}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if pusher.Push(ctx, event) {
		t.Fatal("Push() with canceled context = true, want false")
	}
	if pusher.Push(context.Background(), event) {
		t.Fatal("Push() without target connection = true, want false")
	}
}

func waitForPush(t *testing.T, done <-chan bool) bool {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(time.Second):
		t.Fatal("Push() did not return")
		return false
	}
}
