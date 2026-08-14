package service

import "github.com/prometheus/client_golang/prometheus"

// Metrics 保存 state service 的业务指标。
type Metrics struct {
	RealtimePubsub *prometheus.CounterVec
}

// NewMetrics 创建并注册 state service 业务指标。
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	realtimePubsub := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_state_realtime_pubsub_total",
			Help: "state realtime 发布和订阅操作的累计次数。",
		},
		[]string{"operation", "result"},
	)
	registerer.MustRegister(realtimePubsub)
	return &Metrics{RealtimePubsub: realtimePubsub}
}
