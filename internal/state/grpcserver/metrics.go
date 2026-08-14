package grpcserver

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

// Metrics 保存 state gRPC 服务的业务指标。
type Metrics struct {
	Requests *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

// NewMetrics 创建并注册 state gRPC 业务指标。
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_state_grpc_requests_total",
			Help: "state gRPC 请求的累计次数。",
		},
		[]string{"method", "result"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "game_state_grpc_request_duration_seconds",
			Help:    "state gRPC 请求耗时分布。",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
	registerer.MustRegister(requests, duration)
	return &Metrics{Requests: requests, Duration: duration}
}

// UnaryMetricsInterceptor 统计 state gRPC 一元请求的次数和耗时。
func UnaryMetricsInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startedAt := time.Now()
		response, err := handler(ctx, request)
		if metrics != nil {
			method := metricMethodName(info.FullMethod)
			result := "success"
			if err != nil {
				result = "error"
			}
			metrics.Requests.WithLabelValues(method, result).Inc()
			metrics.Duration.WithLabelValues(method).Observe(time.Since(startedAt).Seconds())
		}
		return response, err
	}
}

func metricMethodName(fullMethod string) string {
	if index := strings.LastIndexByte(fullMethod, '/'); index >= 0 {
		return fullMethod[index+1:]
	}
	return fullMethod
}
