package chat

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsTrackChatAndHistoryOperations(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	repo := &fakeRepository{saved: &Message{MessageKey: "message-1"}}
	service := NewService(ServiceConfig{
		ChatRepository: repo,
		FriendChecker:  &fakeFriendChecker{},
		Metrics:        metrics,
	})

	if _, err := service.SendWorldMessage(context.Background(), SendWorldMessageInput{
		SenderID:         7,
		Content:          "hello",
		ClientMessageKey: "client-1",
	}); err != nil {
		t.Fatalf("valid SendWorldMessage returned error: %v", err)
	}
	if _, err := service.SendWorldMessage(context.Background(), SendWorldMessageInput{SenderID: 7}); err == nil {
		t.Fatal("invalid SendWorldMessage unexpectedly succeeded")
	}
	if _, err := service.ListWorldMessages(context.Background(), ListWorldMessagesInput{PlayerID: 7}); err != nil {
		t.Fatalf("valid ListWorldMessages returned error: %v", err)
	}
	if _, err := service.ListWorldMessages(context.Background(), ListWorldMessagesInput{}); err == nil {
		t.Fatal("invalid ListWorldMessages unexpectedly succeeded")
	}

	if got := chatCounterValue(t, metrics.Messages, "world", "success"); got != 1 {
		t.Fatalf("world message success metric = %v, want 1", got)
	}
	if got := chatCounterValue(t, metrics.Messages, "world", "error"); got != 1 {
		t.Fatalf("world message error metric = %v, want 1", got)
	}
	if got := chatCounterValue(t, metrics.HistoryRequests, "world", "success"); got != 1 {
		t.Fatalf("world history success metric = %v, want 1", got)
	}
	if got := chatCounterValue(t, metrics.HistoryRequests, "world", "error"); got != 1 {
		t.Fatalf("world history error metric = %v, want 1", got)
	}
}

func chatCounterValue(t *testing.T, counter *prometheus.CounterVec, channel, result string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.WithLabelValues(channel, result).Write(metric); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return metric.GetCounter().GetValue()
}
