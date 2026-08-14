package realtime

import (
	"context"
	"net"
	"testing"
	"time"

	"server/internal/contract/realtimepb"
	statecontract "server/internal/contract/state"
	"server/internal/logic/auth"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestHandlerMetricsTracksRealtimeConnections(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	handler := NewHandler(HandlerConfig{
		AuthService:     &fakeHandlerAuth{session: &auth.Session{PlayerID: 7}},
		PresenceService: &fakeHandlerPresence{},
		ServerName:      "logic-test",
		Metrics:         metrics,
	})
	serverConn, clientConn, done := startHandlerSession(t, handler)
	defer serverConn.Close()

	writeClientEnvelope(t, clientConn, authenticateEnvelope(1, "session-token"))
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetAuthenticated().GetPlayerId() != 7 {
		t.Fatalf("authenticated player = %d, want 7", response.GetAuthenticated().GetPlayerId())
	}
	if got := realtimeGaugeValue(t, metrics.Connections); got != 1 {
		t.Fatalf("realtime connections metric after authentication = %v, want 1", got)
	}

	writeClientEnvelope(t, clientConn, heartbeatEnvelope(2))
	heartbeatResponse := readServerEnvelopeOrFail(t, clientConn)
	if heartbeatResponse.GetHeartbeatAck() == nil {
		t.Fatalf("heartbeat response = %v, want heartbeat ack", heartbeatResponse)
	}
	if got := waitForRealtimeCounter(t, metrics.Requests, "heartbeat", "accepted", 1); got != 1 {
		t.Fatalf("heartbeat accepted metric = %v, want 1", got)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitForHandler(t, done)
	if got := realtimeGaugeValue(t, metrics.Connections); got != 0 {
		t.Fatalf("realtime connections metric after disconnect = %v, want 0", got)
	}
}

func TestHandlerMetricsTracksUnsupportedRealtimeRequest(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	handler := NewHandler(HandlerConfig{Metrics: metrics})
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	session := newSession(serverConn)
	done := make(chan bool, 1)
	go func() {
		done <- handler.handleEnvelope(
			context.Background(),
			session,
			&auth.Session{PlayerID: 7},
			1,
			&realtimepb.ClientEnvelope{RequestId: 3},
		)
	}()
	response := readServerEnvelopeOrFail(t, clientConn)
	if response.GetError() == nil {
		t.Fatalf("unsupported response = %v, want error", response)
	}
	if handled := <-done; !handled {
		t.Fatal("unsupported request handler returned false")
	}
	if got := realtimeCounterValue(t, metrics.Requests, "unknown", "accepted"); got != 1 {
		t.Fatalf("unknown request metric = %v, want accepted 1", got)
	}
}

func TestHandlerMetricsTracksLocalRealtimeDelivery(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	handler := NewHandler(HandlerConfig{Metrics: metrics})
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	handler.connManager.Add(8, newSession(serverConn))

	done := make(chan struct{})
	go func() {
		handler.pushRealtimeEvent(context.Background(), &statecontract.RealtimeEvent{
			Type:           statecontract.RealtimeEventFriendRemoved,
			TargetPlayerID: 8,
			ActorPlayerID:  7,
		})
		close(done)
	}()
	envelope := readServerEnvelopeOrFail(t, clientConn)
	<-done
	if envelope.GetFriendRemoved().GetPlayerId() != 7 {
		t.Fatalf("delivery envelope = %v, want friend removed for player 7", envelope)
	}
	if got := realtimeCounterValue(t, metrics.Deliveries, "player", "delivered"); got != 1 {
		t.Fatalf("player delivery metric = %v, want 1", got)
	}
}

func realtimeGaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		t.Fatalf("write gauge metric: %v", err)
	}
	return metric.GetGauge().GetValue()
}

func realtimeCounterValue(t *testing.T, counter *prometheus.CounterVec, label1, label2 string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.WithLabelValues(label1, label2).Write(metric); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return metric.GetCounter().GetValue()
}

func waitForRealtimeCounter(t *testing.T, counter *prometheus.CounterVec, label1, label2 string, want float64) float64 {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		got := realtimeCounterValue(t, counter, label1, label2)
		if got == want || time.Now().After(deadline) {
			return got
		}
		time.Sleep(time.Millisecond)
	}
}
