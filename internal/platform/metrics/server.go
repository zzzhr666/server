package metrics

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ServerConfig 定义 Metrics Server 的监听地址与指标采集源。
type ServerConfig struct {
	Addr     string
	Gatherer prometheus.Gatherer
}

// Server 通过独立的 HTTP Server 暴露 Prometheus 指标。
type Server struct {
	httpServer *http.Server
}

// NewServer 根据配置创建 Metrics Server。
func NewServer(config ServerConfig) (*Server, error) {
	if strings.TrimSpace(config.Addr) == "" || config.Gatherer == nil {
		return nil, ErrInvalidConfig
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(config.Gatherer, promhttp.HandlerOpts{}))

	return &Server{
		httpServer: &http.Server{
			Addr:              config.Addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

// Serve 启动 Metrics HTTP Server，并将正常关闭视为成功。
func (s *Server) Serve() error {
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown 在给定上下文期限内优雅关闭 Metrics HTTP Server。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
