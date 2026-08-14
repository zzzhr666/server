package auth

import "github.com/prometheus/client_golang/prometheus"

// Metrics 保存认证服务的业务指标。
type Metrics struct {
	Attempts *prometheus.CounterVec
}

// NewMetrics 创建并注册认证服务业务指标。
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	attempts := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_logic_auth_attempts_total",
			Help: "认证相关操作的累计次数。",
		},
		[]string{"operation", "result"},
	)
	registerer.MustRegister(attempts)
	return &Metrics{Attempts: attempts}
}
