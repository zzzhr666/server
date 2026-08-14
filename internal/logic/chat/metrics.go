package chat

import "github.com/prometheus/client_golang/prometheus"

// Metrics 保存聊天服务的业务指标。
type Metrics struct {
	Messages        *prometheus.CounterVec
	HistoryRequests *prometheus.CounterVec
}

// NewMetrics 创建并注册聊天服务业务指标。
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	messages := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_logic_chat_messages_total",
			Help: "聊天消息发送的累计次数。",
		},
		[]string{"channel", "result"},
	)
	historyRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_logic_chat_history_requests_total",
			Help: "聊天历史读取请求的累计次数。",
		},
		[]string{"channel", "result"},
	)
	registerer.MustRegister(messages, historyRequests)
	return &Metrics{
		Messages:        messages,
		HistoryRequests: historyRequests,
	}
}
