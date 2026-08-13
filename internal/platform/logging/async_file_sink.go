package logging

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type flushRequest struct {
	done chan error
}

// AsyncFileSink 使用双缓冲将格式化后的日志异步写入文件。
type AsyncFileSink struct {
	file   *os.File
	config FileSinkConfig

	mu           sync.Mutex
	active       []byte
	closed       bool
	pendingBytes int

	flushing       []byte
	flushingOffset int

	wake          chan struct{}
	flushRequests chan flushRequest
	stop          chan struct{}
	done          chan struct{}
	closeOnce     sync.Once

	dropped  atomic.Uint64
	closeErr error
}

// FileSinkConfig 定义异步文件 Sink 的缓冲与刷新策略。
type FileSinkConfig struct {
	Path              string
	InitialBufferSize int
	FlushThreshold    int
	MaxPendingBytes   int
	FlushInterval     time.Duration
}

const (
	defaultInitialBufferSize = 64 * 1024
	defaultFlushThreshold    = 256 * 1024
	defaultMaxPendingBytes   = 8 * 1024 * 1024
	defaultFlushInterval     = 500 * time.Millisecond
)

func normalizeFileSinkConfig(config FileSinkConfig) FileSinkConfig {
	if config.InitialBufferSize <= 0 {
		config.InitialBufferSize = defaultInitialBufferSize
	}
	if config.FlushThreshold <= 0 {
		config.FlushThreshold = defaultFlushThreshold
	}
	if config.MaxPendingBytes <= 0 {
		config.MaxPendingBytes = defaultMaxPendingBytes
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
	}
	return config
}

func validateFileSinkConfig(config FileSinkConfig) error {
	switch {
	case config.Path == "":
		return errors.New("logging: file sink path is empty")
	case config.FlushThreshold < config.InitialBufferSize:
		return errors.New("logging: flush threshold must not be smaller than initial buffer size")
	case config.MaxPendingBytes < config.FlushThreshold:
		return errors.New("logging: max pending bytes must not be smaller than flush threshold")
	default:
		return nil
	}
}

// NewAsyncFileSink 创建异步文件 Sink 并启动后台写入协程。
func NewAsyncFileSink(config FileSinkConfig) (*AsyncFileSink, error) {
	config = normalizeFileSinkConfig(config)
	if err := validateFileSinkConfig(config); err != nil {
		return nil, err
	}
	directory := filepath.Dir(config.Path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(config.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	sink := AsyncFileSink{
		file:          file,
		config:        config,
		active:        make([]byte, 0, config.InitialBufferSize),
		flushing:      make([]byte, 0, config.InitialBufferSize),
		wake:          make(chan struct{}, 1),
		flushRequests: make(chan flushRequest),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go sink.run()
	return &sink, nil
}

// Log 将一条日志追加到活动缓冲区，超过内存上限时丢弃整条日志。
func (s *AsyncFileSink) Log(message string) {
	requiredBytes := len(message) + 1
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return
	}
	if requiredBytes > s.config.MaxPendingBytes-s.pendingBytes {
		s.dropped.Add(1)
		s.mu.Unlock()
		return
	}
	s.active = append(s.active, message...)
	s.active = append(s.active, '\n')
	s.pendingBytes += requiredBytes
	shouldWake := len(s.active) >= s.config.FlushThreshold

	s.mu.Unlock()
	if shouldWake {
		s.notifyWriter()
	}

}

// Dropped 返回因待写数据达到内存上限而被丢弃的日志数量。
func (s *AsyncFileSink) Dropped() uint64 {
	return s.dropped.Load()
}

// Close 停止接收日志，排空缓冲区并关闭日志文件。
func (s *AsyncFileSink) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.stop)
		s.mu.Unlock()
	})
	<-s.done
	return s.closeErr
}

// Flush 将当前待写日志同步到持久化存储。
func (s *AsyncFileSink) Flush() error {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()

	if closed {
		<-s.done
		return s.closeErr
	}
	request := flushRequest{
		done: make(chan error, 1),
	}
	select {
	case s.flushRequests <- request:
		return <-request.done
	case <-s.stop:
		<-s.done
		return s.closeErr
	}
}

func (s *AsyncFileSink) notifyWriter() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *AsyncFileSink) writeFlushing() error {
	for s.flushingOffset < len(s.flushing) {
		written, err := s.file.Write(s.flushing[s.flushingOffset:])
		if written > 0 {
			s.flushingOffset += written
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	if len(s.flushing) == 0 {
		return nil
	}
	flushedBytes := len(s.flushing)
	s.flushing = s.flushing[:0]
	s.flushingOffset = 0

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingBytes -= flushedBytes
	return nil
}

func (s *AsyncFileSink) flushPending() error {
	if err := s.writeFlushing(); err != nil {
		return err
	}
	s.mu.Lock()
	if len(s.active) == 0 {
		s.mu.Unlock()
		return nil
	}
	s.active, s.flushing = s.flushing[:0], s.active
	s.flushingOffset = 0
	s.mu.Unlock()
	return s.writeFlushing()
}

func (s *AsyncFileSink) run() {
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()
	defer close(s.done)
	for {
		select {
		case <-s.wake:
			_ = s.flushPending()
		case <-ticker.C:
			_ = s.flushPending()
		case request := <-s.flushRequests:
			err := s.flushPending()
			if err == nil {
				err = s.file.Sync()
			}
			request.done <- err
		case <-s.stop:
			s.closeErr = s.shutdown()
			return
		}
	}
}

func (s *AsyncFileSink) shutdown() error {
	flushErr := s.flushPending()
	syncErr := s.file.Sync()
	closeErr := s.file.Close()
	return errors.Join(flushErr, syncErr, closeErr)
}
