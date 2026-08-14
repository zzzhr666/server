package grpcserver

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryMetricsInterceptorRecordsRequestsAndDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	interceptor := UnaryMetricsInterceptor(metrics)
	info := &grpc.UnaryServerInfo{FullMethod: "/state.v1.StateService/GetPlayer"}

	if _, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("successful interceptor call returned error: %v", err)
	}
	if _, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return nil, status.Error(codes.NotFound, "missing")
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("failed interceptor call error = %v, want not found", err)
	}

	if got := grpcCounterValue(t, metrics.Requests, "GetPlayer", "success"); got != 1 {
		t.Fatalf("successful request metric = %v, want 1", got)
	}
	if got := grpcCounterValue(t, metrics.Requests, "GetPlayer", "error"); got != 1 {
		t.Fatalf("failed request metric = %v, want 1", got)
	}
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "game_state_grpc_request_duration_seconds" {
			continue
		}
		if len(family.GetMetric()) != 1 || family.GetMetric()[0].GetHistogram().GetSampleCount() != 2 {
			t.Fatalf("duration metrics = %v, want one method with two samples", family.GetMetric())
		}
		return
	}
	t.Fatal("duration metric family was not gathered")
}

func grpcCounterValue(t *testing.T, counter *prometheus.CounterVec, method, result string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.WithLabelValues(method, result).Write(metric); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return metric.GetCounter().GetValue()
}
