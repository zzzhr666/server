package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDefaultLoggerDebugUsesConsoleAndFileSinks(t *testing.T) {
	logDir := t.TempDir()
	logger, err := NewDefaultLogger(DefaultLoggerOptions{
		Level:       DebugLevel,
		Mode:        DebugMode,
		ServiceName: "logic-server",
		LogDir:      logDir,
	})
	if err != nil {
		t.Fatalf("NewDefaultLogger() error = %v", err)
	}

	if len(logger.sinks) != 2 {
		t.Fatalf("sink count = %d, want 2", len(logger.sinks))
	}
	if _, ok := logger.sinks[0].(*ConsoleSink); !ok {
		t.Fatalf("sink 0 type = %T, want *ConsoleSink", logger.sinks[0])
	}
	if _, ok := logger.sinks[1].(*AsyncFileSink); !ok {
		t.Fatalf("sink 1 type = %T, want *AsyncFileSink", logger.sinks[1])
	}

	logger.Info("debug mode message")
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	contents, err := os.ReadFile(filepath.Join(logDir, "logic-server.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(contents), "debug mode message") {
		t.Fatalf("log contents = %q, want message", contents)
	}
}

func TestNewDefaultLoggerReleaseUsesFileSinkOnly(t *testing.T) {
	logDir := t.TempDir()
	logger, err := NewDefaultLogger(DefaultLoggerOptions{
		Level:       InfoLevel,
		Mode:        ReleaseMode,
		ServiceName: "state-server",
		LogDir:      logDir,
	})
	if err != nil {
		t.Fatalf("NewDefaultLogger() error = %v", err)
	}
	if len(logger.sinks) != 1 {
		t.Fatalf("sink count = %d, want 1", len(logger.sinks))
	}
	if _, ok := logger.sinks[0].(*AsyncFileSink); !ok {
		t.Fatalf("sink type = %T, want *AsyncFileSink", logger.sinks[0])
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNewDefaultLoggerRejectsMissingServiceName(t *testing.T) {
	if _, err := NewDefaultLogger(DefaultLoggerOptions{Mode: DebugMode}); err == nil {
		t.Fatal("NewDefaultLogger() error = nil, want an error")
	}
}
