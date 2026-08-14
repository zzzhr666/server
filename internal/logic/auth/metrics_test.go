package auth

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsTrackAuthenticationAttempts(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	repo := newFakeAuthRepository()
	service := NewService(ServiceConfig{
		AuthRepository: repo,
		PlayerService:  newFakePlayerService(),
		SessionTTL:     time.Hour,
		Metrics:        metrics,
	})

	if _, err := service.Register(context.Background(), RegisterInput{}); err == nil {
		t.Fatal("invalid Register unexpectedly succeeded")
	}
	if _, err := service.Register(context.Background(), RegisterInput{
		Username:      "alice",
		PlainPassword: "password123",
		Nickname:      "Alice",
	}); err != nil {
		t.Fatalf("valid Register returned error: %v", err)
	}
	if _, err := service.Login(context.Background(), LoginInput{
		Username:      "alice",
		PlainPassword: "wrong-password",
	}); err == nil {
		t.Fatal("invalid Login unexpectedly succeeded")
	}
	if err := service.Logout(context.Background(), ""); err == nil {
		t.Fatal("invalid Logout unexpectedly succeeded")
	}

	if got := authCounterValue(t, metrics.Attempts, "register", "error"); got != 1 {
		t.Fatalf("register error metric = %v, want 1", got)
	}
	if got := authCounterValue(t, metrics.Attempts, "register", "success"); got != 1 {
		t.Fatalf("register success metric = %v, want 1", got)
	}
	if got := authCounterValue(t, metrics.Attempts, "login", "error"); got != 1 {
		t.Fatalf("login error metric = %v, want 1", got)
	}
	if got := authCounterValue(t, metrics.Attempts, "logout", "error"); got != 1 {
		t.Fatalf("logout error metric = %v, want 1", got)
	}
}

func authCounterValue(t *testing.T, counter *prometheus.CounterVec, operation, result string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.WithLabelValues(operation, result).Write(metric); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return metric.GetCounter().GetValue()
}
