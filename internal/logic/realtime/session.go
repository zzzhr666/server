package realtime

import (
	"net"
	"server/internal/contract/realtimepb"
	"sync"
	"time"
)

const sessionWriteTimeout = 3 * time.Second

type session struct {
	mu       sync.Mutex
	once     sync.Once
	conn     net.Conn
	closeErr error
}

func newSession(conn net.Conn) *session {
	return &session{
		conn: conn,
	}
}

func (s *session) Close() error {
	s.once.Do(func() {
		s.closeErr = s.conn.Close()
	})
	return s.closeErr
}

func (s *session) Read() (*realtimepb.ClientEnvelope, error) {
	//由于只有一个读协程可以调用此方法，因此不加锁
	return readClientEnvelope(s.conn)
}

func (s *session) Write(e *realtimepb.ServerEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.SetWriteDeadline(time.Now().Add(sessionWriteTimeout)); err != nil {
		return err
	}
	defer func() {
		_ = s.conn.SetWriteDeadline(time.Time{})
	}()
	return writeServerEnvelope(s.conn, e)
}

func (s *session) setReadDeadline(deadline time.Time) error {
	return s.conn.SetReadDeadline(deadline)
}
