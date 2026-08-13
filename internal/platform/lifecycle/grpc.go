package lifecycle

import (
	"context"
	"fmt"
)

type grpcServer interface {
	GracefulStop()
	Stop()
}

// GracefulStopGRPC 等待 gRPC 请求完成，并在 Context 到期后强制停止服务。
func GracefulStopGRPC(ctx context.Context, server grpcServer) error {
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		server.Stop()
		<-done
		return fmt.Errorf("gracefully stop gRPC server: %w", ctx.Err())
	}
}
