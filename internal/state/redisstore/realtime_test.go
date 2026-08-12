package redisstore

import (
	statecontract "server/internal/contract/state"
	"testing"
)

func TestValidRealtimeRoute(t *testing.T) {
	tests := []struct {
		name  string
		route statecontract.RealtimeRoute
		want  bool
	}{
		{
			name: "server route",
			route: statecontract.RealtimeRoute{
				Type:       statecontract.RealtimeRouteServer,
				ServerName: "logic-1",
			},
			want: true,
		},
		{
			name: "broadcast route",
			route: statecontract.RealtimeRoute{
				Type: statecontract.RealtimeRouteBroadcast,
			},
			want: true,
		},
		{
			name: "server route without server name",
			route: statecontract.RealtimeRoute{
				Type: statecontract.RealtimeRouteServer,
			},
			want: false,
		},
		{
			name: "broadcast route with server name",
			route: statecontract.RealtimeRoute{
				Type:       statecontract.RealtimeRouteBroadcast,
				ServerName: "logic-1",
			},
			want: false,
		},
		{
			name:  "unspecified route",
			route: statecontract.RealtimeRoute{},
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validRealtimeRoute(test.route); got != test.want {
				t.Fatalf("validRealtimeRoute(%+v) = %t, want %t", test.route, got, test.want)
			}
		})
	}
}

func TestValidRealtimeDelivery(t *testing.T) {
	event := &statecontract.RealtimeEvent{
		Type:           statecontract.RealtimeEventChatMessage,
		TargetPlayerID: 8,
	}

	tests := []struct {
		name     string
		delivery *statecontract.RealtimeDelivery
		want     bool
	}{
		{
			name: "complete server delivery",
			delivery: &statecontract.RealtimeDelivery{
				Route: statecontract.RealtimeRoute{
					Type:       statecontract.RealtimeRouteServer,
					ServerName: "logic-1",
				},
				Event: event,
			},
			want: true,
		},
		{name: "nil delivery", delivery: nil, want: false},
		{
			name: "nil event",
			delivery: &statecontract.RealtimeDelivery{
				Route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteServer, ServerName: "logic-1"},
			},
			want: false,
		},
		{
			name: "missing target player",
			delivery: &statecontract.RealtimeDelivery{
				Route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteServer, ServerName: "logic-1"},
				Event: &statecontract.RealtimeEvent{Type: statecontract.RealtimeEventChatMessage},
			},
			want: false,
		},
		{
			name: "broadcast delivery",
			delivery: &statecontract.RealtimeDelivery{
				Route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteBroadcast},
				Event: event,
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validRealtimeDelivery(test.delivery); got != test.want {
				t.Fatalf("validRealtimeDelivery(%+v) = %t, want %t", test.delivery, got, test.want)
			}
		})
	}
}

func TestRealtimeDeliveryChannel(t *testing.T) {
	tests := []struct {
		name  string
		route statecontract.RealtimeRoute
		want  string
		ok    bool
	}{
		{name: "server", route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteServer, ServerName: "logic-1"}, want: realtimeChannelKey("logic-1"), ok: true},
		{name: "broadcast", route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteBroadcast}, want: realtimeChannelKey(statecontract.RealtimeBroadcastChannelName), ok: true},
		{name: "unknown route", route: statecontract.RealtimeRoute{Type: "room", ServerName: "room-1"}, want: "", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := realtimeDeliveryChannel(test.route)
			if got != test.want || ok != test.ok {
				t.Fatalf("realtimeDeliveryChannel(%+v) = (%q, %t), want (%q, %t)", test.route, got, ok, test.want, test.ok)
			}
		})
	}
}
