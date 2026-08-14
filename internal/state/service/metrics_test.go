package service

import (
	"context"
	"errors"
	"testing"

	statecontract "server/internal/contract/state"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsTrackRealtimePubsubOperations(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	realtime := &fakeRealtimeStore{subscribeDeliveries: make(chan *statecontract.RealtimeDelivery)}
	svc := NewService(StoreConfig{Realtime: realtime, Metrics: metrics})
	delivery := &statecontract.RealtimeDelivery{
		Route: statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteBroadcast},
		Event: &statecontract.RealtimeEvent{Type: statecontract.RealtimeEventChatMessage},
	}

	if err := svc.PublishRealtime(context.Background(), delivery); err != nil {
		t.Fatalf("successful PublishRealtime returned error: %v", err)
	}
	realtime.publishErr = errors.New("publish failed")
	if err := svc.PublishRealtime(context.Background(), delivery); err == nil {
		t.Fatal("failed PublishRealtime unexpectedly succeeded")
	}
	if _, err := svc.SubscribeRealtime(context.Background(), statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteBroadcast}); err != nil {
		t.Fatalf("successful SubscribeRealtime returned error: %v", err)
	}
	realtime.subscribeErr = errors.New("subscribe failed")
	if _, err := svc.SubscribeRealtime(context.Background(), statecontract.RealtimeRoute{Type: statecontract.RealtimeRouteBroadcast}); err == nil {
		t.Fatal("failed SubscribeRealtime unexpectedly succeeded")
	}

	if got := stateCounterValue(t, metrics.RealtimePubsub, "publish", "success"); got != 1 {
		t.Fatalf("publish success metric = %v, want 1", got)
	}
	if got := stateCounterValue(t, metrics.RealtimePubsub, "publish", "error"); got != 1 {
		t.Fatalf("publish error metric = %v, want 1", got)
	}
	if got := stateCounterValue(t, metrics.RealtimePubsub, "subscribe", "success"); got != 1 {
		t.Fatalf("subscribe success metric = %v, want 1", got)
	}
	if got := stateCounterValue(t, metrics.RealtimePubsub, "subscribe", "error"); got != 1 {
		t.Fatalf("subscribe error metric = %v, want 1", got)
	}
}

func stateCounterValue(t *testing.T, counter *prometheus.CounterVec, operation, result string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.WithLabelValues(operation, result).Write(metric); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return metric.GetCounter().GetValue()
}

type fakeRealtimeStore struct {
	subscribeDeliveries chan *statecontract.RealtimeDelivery
	publishErr          error
	subscribeErr        error
}

func (f *fakeRealtimeStore) PublishRealtime(context.Context, *statecontract.RealtimeDelivery) error {
	return f.publishErr
}

func (f *fakeRealtimeStore) SubscribeRealtime(context.Context, statecontract.RealtimeRoute) (<-chan *statecontract.RealtimeDelivery, error) {
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	return f.subscribeDeliveries, nil
}
