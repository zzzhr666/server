package realtime

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"server/internal/contract/realtimepb"

	"google.golang.org/protobuf/proto"
)

func TestSessionRead(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	want := &realtimepb.ClientEnvelope{
		RequestId: 1,
		Payload: &realtimepb.ClientEnvelope_Heartbeat{
			Heartbeat: &realtimepb.HeartbeatRequest{},
		},
	}
	writeErr := make(chan error, 1)
	go func() {
		payload, err := proto.Marshal(want)
		if err == nil {
			err = writeFrame(clientConn, payload)
		}
		writeErr <- err
	}()

	got, err := newSession(serverConn).Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("client envelope = %v, want %v", got, want)
	}
	if err := <-writeErr; err != nil {
		t.Fatalf("client write error = %v", err)
	}
}

func TestSessionWriteSerializesConcurrentFrames(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	const messageCount = 16
	if err := serverConn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set write deadline: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	receivedIDs := make(chan uint64, messageCount)
	readErr := make(chan error, 1)
	go func() {
		for range messageCount {
			envelope, err := readServerEnvelope(clientConn)
			if err != nil {
				readErr <- err
				return
			}
			receivedIDs <- envelope.GetRequestId()
		}
		readErr <- nil
	}()

	session := newSession(serverConn)
	start := make(chan struct{})
	writeErrs := make(chan error, messageCount)
	var writers sync.WaitGroup
	for id := uint64(1); id <= messageCount; id++ {
		writers.Add(1)
		go func(requestID uint64) {
			defer writers.Done()
			<-start
			writeErrs <- session.Write(&realtimepb.ServerEnvelope{
				RequestId: requestID,
				Payload: &realtimepb.ServerEnvelope_HeartbeatAck{
					HeartbeatAck: &realtimepb.HeartbeatAck{},
				},
			})
		}(id)
	}

	close(start)
	writers.Wait()
	close(writeErrs)
	for err := range writeErrs {
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := <-readErr; err != nil {
		t.Fatalf("read server envelope: %v", err)
	}

	seen := make(map[uint64]bool, messageCount)
	for range messageCount {
		requestID := <-receivedIDs
		if seen[requestID] {
			t.Fatalf("duplicate request ID %d", requestID)
		}
		seen[requestID] = true
	}
	for requestID := uint64(1); requestID <= messageCount; requestID++ {
		if !seen[requestID] {
			t.Fatalf("request ID %d was not received", requestID)
		}
	}
}

func TestSessionCloseClosesConnectionOnce(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	conn := &closeCountingConn{Conn: serverConn}
	session := newSession(conn)

	const closerCount = 16
	errs := make(chan error, closerCount)
	var closers sync.WaitGroup
	for range closerCount {
		closers.Add(1)
		go func() {
			defer closers.Done()
			errs <- session.Close()
		}()
	}
	closers.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}
	if got := conn.closeCalls.Load(); got != 1 {
		t.Fatalf("underlying Close calls = %d, want 1", got)
	}
}

func TestSessionWriteAppliesAndClearsDeadline(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	conn := &deadlineRecordingConn{Conn: serverConn}
	session := newSession(conn)
	readErr := make(chan error, 1)
	go func() {
		_, err := readServerEnvelope(clientConn)
		readErr <- err
	}()

	if err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: 1,
		Payload: &realtimepb.ServerEnvelope_HeartbeatAck{
			HeartbeatAck: &realtimepb.HeartbeatAck{},
		},
	}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := <-readErr; err != nil {
		t.Fatalf("read server envelope: %v", err)
	}

	deadlines := conn.writeDeadlines()
	if got := len(deadlines); got != 2 {
		t.Fatalf("SetWriteDeadline calls = %d, want 2", got)
	}
	if deadlines[0].IsZero() {
		t.Fatal("write deadline = zero, want a bounded deadline")
	}
	if !deadlines[1].IsZero() {
		t.Fatalf("cleared write deadline = %v, want zero", deadlines[1])
	}
}

func TestSessionWriteTimesOutWhenPeerStopsReading(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	session := newSession(&shortWriteDeadlineConn{Conn: serverConn})

	startedAt := time.Now()
	err := session.Write(&realtimepb.ServerEnvelope{
		RequestId: 1,
		Payload: &realtimepb.ServerEnvelope_HeartbeatAck{
			HeartbeatAck: &realtimepb.HeartbeatAck{},
		},
	})
	if err == nil {
		t.Fatal("Write() error = nil, want timeout")
	}
	networkErr, ok := err.(net.Error)
	if !ok || !networkErr.Timeout() {
		t.Fatalf("Write() error = %v, want timeout error", err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("Write() blocked for %v, want less than one second", elapsed)
	}
}

func readServerEnvelope(reader net.Conn) (*realtimepb.ServerEnvelope, error) {
	payload, err := readFrame(reader)
	if err != nil {
		return nil, err
	}
	var envelope realtimepb.ServerEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

type closeCountingConn struct {
	net.Conn
	closeCalls atomic.Int32
}

func (c *closeCountingConn) Close() error {
	c.closeCalls.Add(1)
	return c.Conn.Close()
}

type deadlineRecordingConn struct {
	net.Conn
	mu        sync.Mutex
	deadlines []time.Time
}

func (c *deadlineRecordingConn) SetWriteDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.deadlines = append(c.deadlines, deadline)
	c.mu.Unlock()
	return c.Conn.SetWriteDeadline(deadline)
}

func (c *deadlineRecordingConn) writeDeadlines() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Time(nil), c.deadlines...)
}

type shortWriteDeadlineConn struct {
	net.Conn
}

func (c *shortWriteDeadlineConn) SetWriteDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		return c.Conn.SetWriteDeadline(time.Time{})
	}
	return c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
}
