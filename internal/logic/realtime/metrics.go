package realtime

import "github.com/prometheus/client_golang/prometheus"

// Metrics 保存 logic realtime 的业务指标。
type Metrics struct {
	Connections prometheus.Gauge
	Requests    *prometheus.CounterVec
	Deliveries  *prometheus.CounterVec
}

// NewMetrics 创建并注册 logic realtime 业务指标。
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	connections := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_logic_realtime_connections",
		Help: "当前逻辑服务器的实时连接数。",
	})
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_logic_realtime_requests_total",
			Help: "实时协议请求被连接处理的累计次数。",
		},
		[]string{"type", "result"},
	)
	deliveries := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_logic_realtime_deliveries_total",
			Help: "实时事件向本机客户端投递的累计次数。",
		},
		[]string{"route", "result"},
	)
	registerer.MustRegister(connections, requests, deliveries)
	return &Metrics{
		Connections: connections,
		Requests:    requests,
		Deliveries:  deliveries,
	}
}

func (m *Metrics) observeRequest(requestType string, success bool) {
	if m == nil {
		return
	}
	result := "accepted"
	if !success {
		result = "closed"
	}
	m.Requests.WithLabelValues(requestType, result).Inc()
}

func (m *Metrics) observeDelivery(route string, success bool) {
	if m == nil {
		return
	}
	result := "delivered"
	if !success {
		result = "failed"
	}
	m.Deliveries.WithLabelValues(route, result).Inc()
}
