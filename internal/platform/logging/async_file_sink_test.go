package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNormalizeFileSinkConfigUsesDefaultsForNonPositiveValues(t *testing.T) {
	config := normalizeFileSinkConfig(FileSinkConfig{
		Path:              "server.log",
		InitialBufferSize: -1,
		FlushThreshold:    0,
		MaxPendingBytes:   -1,
		FlushInterval:     0,
	})

	if config.InitialBufferSize != defaultInitialBufferSize {
		t.Fatalf("InitialBufferSize = %d, want %d", config.InitialBufferSize, defaultInitialBufferSize)
	}
	if config.FlushThreshold != defaultFlushThreshold {
		t.Fatalf("FlushThreshold = %d, want %d", config.FlushThreshold, defaultFlushThreshold)
	}
	if config.MaxPendingBytes != defaultMaxPendingBytes {
		t.Fatalf("MaxPendingBytes = %d, want %d", config.MaxPendingBytes, defaultMaxPendingBytes)
	}
	if config.FlushInterval != defaultFlushInterval {
		t.Fatalf("FlushInterval = %s, want %s", config.FlushInterval, defaultFlushInterval)
	}
}

func TestNewAsyncFileSinkRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config FileSinkConfig
	}{
		{
			name: "empty path",
			config: FileSinkConfig{
				InitialBufferSize: 1,
				FlushThreshold:    1,
				MaxPendingBytes:   1,
			},
		},
		{
			name: "threshold smaller than initial buffer",
			config: FileSinkConfig{
				Path:              filepath.Join(t.TempDir(), "server.log"),
				InitialBufferSize: 2,
				FlushThreshold:    1,
				MaxPendingBytes:   2,
			},
		},
		{
			name: "pending limit smaller than threshold",
			config: FileSinkConfig{
				Path:              filepath.Join(t.TempDir(), "server.log"),
				InitialBufferSize: 1,
				FlushThreshold:    2,
				MaxPendingBytes:   1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sink, err := NewAsyncFileSink(test.config)
			if err == nil {
				_ = sink.Close()
				t.Fatal("NewAsyncFileSink() error = nil, want an error")
			}
		})
	}
}

func TestAsyncFileSinkFlushWritesMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "server.log")
	sink := newTestAsyncFileSink(t, FileSinkConfig{
		Path:              path,
		InitialBufferSize: 8,
		FlushThreshold:    1024,
		MaxPendingBytes:   4096,
		FlushInterval:     time.Hour,
	})

	sink.Log("first")
	sink.Log("second")
	if err := sink.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if got, want := readLogFile(t, path), "first\nsecond\n"; got != want {
		t.Fatalf("log contents = %q, want %q", got, want)
	}
}

func TestAsyncFileSinkFlushesAtThreshold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	sink := newTestAsyncFileSink(t, FileSinkConfig{
		Path:              path,
		InitialBufferSize: 1,
		FlushThreshold:    6,
		MaxPendingBytes:   1024,
		FlushInterval:     time.Hour,
	})

	sink.Log("hello")
	waitForLogContents(t, path, "hello\n")
}

func TestAsyncFileSinkFlushesOnInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	sink := newTestAsyncFileSink(t, FileSinkConfig{
		Path:              path,
		InitialBufferSize: 1,
		FlushThreshold:    1024,
		MaxPendingBytes:   2048,
		FlushInterval:     10 * time.Millisecond,
	})

	sink.Log("periodic")
	waitForLogContents(t, path, "periodic\n")
}

func TestAsyncFileSinkCloseDrainsAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	sink, err := NewAsyncFileSink(FileSinkConfig{
		Path:              path,
		InitialBufferSize: 1,
		FlushThreshold:    1024,
		MaxPendingBytes:   2048,
		FlushInterval:     time.Hour,
	})
	if err != nil {
		t.Fatalf("NewAsyncFileSink() error = %v", err)
	}
	sink.Log("before close")

	const callers = 8
	errors := make(chan error, callers)
	var waitGroup sync.WaitGroup
	for range callers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			errors <- sink.Close()
		}()
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}

	sink.Log("after close")
	if err := sink.Flush(); err != nil {
		t.Fatalf("Flush() after Close() error = %v", err)
	}
	if got, want := readLogFile(t, path), "before close\n"; got != want {
		t.Fatalf("log contents = %q, want %q", got, want)
	}
}

func TestAsyncFileSinkDropsWholeMessageAtMemoryLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	sink := newTestAsyncFileSink(t, FileSinkConfig{
		Path:              path,
		InitialBufferSize: 1,
		FlushThreshold:    16,
		MaxPendingBytes:   16,
		FlushInterval:     time.Hour,
	})

	sink.Log("1234567890")
	sink.Log("abcdefghij")
	if err := sink.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if got := sink.Dropped(); got != 1 {
		t.Fatalf("dropped = %d, want 1", got)
	}
	if got, want := readLogFile(t, path), "1234567890\n"; got != want {
		t.Fatalf("log contents = %q, want %q", got, want)
	}
}

func TestAsyncFileSinkWritesConcurrently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	sink := newTestAsyncFileSink(t, FileSinkConfig{
		Path:              path,
		InitialBufferSize: 64,
		FlushThreshold:    256,
		MaxPendingBytes:   1024 * 1024,
		FlushInterval:     time.Hour,
	})

	const writers = 16
	const messagesPerWriter = 100
	var waitGroup sync.WaitGroup
	for writer := range writers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for message := range messagesPerWriter {
				sink.Log(fmt.Sprintf("writer-%02d-message-%03d", writer, message))
			}
		}()
	}
	waitGroup.Wait()
	if err := sink.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(readLogFile(t, path), "\n"), "\n")
	sort.Strings(lines)
	want := make([]string, 0, writers*messagesPerWriter)
	for writer := range writers {
		for message := range messagesPerWriter {
			want = append(want, fmt.Sprintf("writer-%02d-message-%03d", writer, message))
		}
	}
	sort.Strings(want)
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("written messages do not match: got %d lines, want %d", len(lines), len(want))
	}
}

func newTestAsyncFileSink(t *testing.T, config FileSinkConfig) *AsyncFileSink {
	t.Helper()
	sink, err := NewAsyncFileSink(config)
	if err != nil {
		t.Fatalf("NewAsyncFileSink() error = %v", err)
	}
	t.Cleanup(func() {
		if err := sink.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return sink
}

func waitForLogContents(t *testing.T, path string, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(path)
		if err == nil && string(contents) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("log contents = %q, want %q", readLogFile(t, path), want)
}

func readLogFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(contents)
}
