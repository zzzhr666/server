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
	listener  net.Listener
	handler   *Handler
	once      sync.Once
	closeErr  error
	mu        sync.Mutex
	closed    bool
	sessions  map[*session]struct{}
	sessionWG sync.WaitGroup
}

// NewServer 使用指定监听器和处理器创建 TCP 服务。
func NewServer(listener net.Listener, handler *Handler) *Server {
	return &Server{
		listener: listener,
		handler:  handler,
		sessions: make(map[*session]struct{}),
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
			s.sessionWG.Wait()
			return nil
		} else if err != nil {
			logging.Error("realtime tcp accept failed: %v", err)
			s.Close()
			s.sessionWG.Wait()
			return err
		}
		logging.Debug("realtime tcp connection accepted remote=%s", conn.RemoteAddr())

		curSession := newSession(conn)
		if !s.registerSession(curSession) {
			logging.Error("realtime tcp session register failed: server has been closed")
			_ = conn.Close()
			s.sessionWG.Wait()
			return nil
		}
		go func(curSession *session) {
			defer func() {
				_ = curSession.Close()
				s.unregisterSession(curSession)
			}()
			s.handler.serveSession(ctx, curSession)
		}(curSession)
	}
}

func (s *Server) registerSession(session *session) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.sessions[session] = struct{}{}
	s.sessionWG.Add(1)
	return true
}

func (s *Server) unregisterSession(session *session) {
	s.mu.Lock()
	delete(s.sessions, session)
	s.mu.Unlock()
	s.sessionWG.Done()
}

// Close 停止 TCP 监听器。
func (s *Server) Close() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		sessions := make([]*session, 0, len(s.sessions))
		for current := range s.sessions {
			sessions = append(sessions, current)
		}
		s.mu.Unlock()
		s.closeErr = s.listener.Close()
		if s.closeErr != nil {
			logging.Error("realtime tcp listener close failed: %v", s.closeErr)
		} else {
			logging.Info("realtime tcp server stopped")
		}
		for _, current := range sessions {
			_ = current.Close()
		}
	})
}
