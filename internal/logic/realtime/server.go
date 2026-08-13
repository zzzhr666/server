package realtime

import (
	"context"
	"errors"
	"net"
	"server/internal/platform/logging"
	"sync"
)

// Server 接受 TCP 连接并交由 Handler 处理。
type Server struct {
	listener net.Listener
	handler  *Handler
	once     sync.Once
	closeErr error
}

// NewServer 使用指定监听器和处理器创建 TCP 服务。
func NewServer(listener net.Listener, handler *Handler) *Server {
	return &Server{
		listener: listener,
		handler:  handler,
	}
}

// Serve 接受 TCP 连接，直到 context 取消或监听器关闭。
func (s *Server) Serve(ctx context.Context) error {
	logging.Info("realtime tcp server serving")
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-done:
		}
	}()
	defer close(done)
	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return nil
		} else if err != nil {
			logging.Error("realtime tcp accept failed: %v", err)
			return err
		}
		logging.Debug("realtime tcp connection accepted remote=%s", conn.RemoteAddr())

		go func(conn net.Conn) {
			newSession := session{
				conn: conn,
			}
			defer func() {
				_ = newSession.Close()
			}()
			s.handler.serveSession(ctx, &newSession)
		}(conn)
	}
}

// Close 停止 TCP 监听器。
func (s *Server) Close() {
	s.once.Do(func() {
		s.closeErr = s.listener.Close()
		if s.closeErr != nil {
			logging.Error("realtime tcp listener close failed: %v", s.closeErr)
		} else {
			logging.Info("realtime tcp server stopped")
		}
	})
}
