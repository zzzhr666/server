package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry 封装进程内独立的 Prometheus 指标注册表。
type Registry struct {
	inner *prometheus.Registry
}

// NewRegistry 创建并注册 Go 运行时与当前进程指标。
func NewRegistry() *Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return &Registry{
		inner: registry,
	}
}

// Registerer 返回用于注册业务指标的注册接口。
func (r *Registry) Registerer() prometheus.Registerer {
	return r.inner
}

// Gatherer 返回用于采集当前注册表指标的采集接口。
func (r *Registry) Gatherer() prometheus.Gatherer {
	return r.inner
}
