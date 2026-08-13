package logging

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestLoggerFormatsRecordAndDistributesToEverySink(t *testing.T) {
	formatter := &recordingFormatter{}
	firstSink := &recordingSink{}
	secondSink := &recordingSink{}
	logger := NewLogger(LoggerConfig{
		MinLevel:  InfoLevel,
		Formatter: formatter,
		Sinks:     []Sink{firstSink, secondSink},
	})

	_, wantFile, wantLine, _ := runtime.Caller(0)
	logger.Info("player %d joined %s", 7, "room-1")
	wantLine++

	if formatter.calls != 1 {
		t.Fatalf("formatter calls = %d, want 1", formatter.calls)
	}
	if formatter.record.Level != InfoLevel {
		t.Fatalf("record level = %v, want %v", formatter.record.Level, InfoLevel)
	}
	if formatter.record.Message != "player 7 joined room-1" {
		t.Fatalf("record message = %q, want formatted message", formatter.record.Message)
	}
	if formatter.record.File != wantFile || formatter.record.Line != wantLine {
		t.Fatalf("record source = %s:%d, want %s:%d", formatter.record.File, formatter.record.Line, wantFile, wantLine)
	}
	if !strings.HasSuffix(formatter.record.Function, ".TestLoggerFormatsRecordAndDistributesToEverySink") {
		t.Fatalf("record function = %q, want test function", formatter.record.Function)
	}
	if formatter.record.Time.IsZero() {
		t.Fatal("record time is zero")
	}

	for index, sink := range []*recordingSink{firstSink, secondSink} {
		if len(sink.messages) != 1 || sink.messages[0] != "formatted record" {
			t.Fatalf("sink %d messages = %v, want one formatted record", index, sink.messages)
		}
	}
}

func TestLoggerFiltersBeforeFormatting(t *testing.T) {
	formatter := &recordingFormatter{}
	sink := &recordingSink{}
	logger := NewLogger(LoggerConfig{
		MinLevel:  InfoLevel,
		Formatter: formatter,
		Sinks:     []Sink{sink},
	})

	logger.Debug("filtered message %d", 7)

	if formatter.calls != 0 {
		t.Fatalf("formatter calls = %d, want 0", formatter.calls)
	}
	if len(sink.messages) != 0 {
		t.Fatalf("sink messages = %v, want none", sink.messages)
	}
}

func TestLoggerFlushesEverySupportedSinkAndJoinsErrors(t *testing.T) {
	firstErr := errors.New("first flush failed")
	secondErr := errors.New("second flush failed")
	firstSink := &lifecycleSink{flushErr: firstErr}
	plainSink := &recordingSink{}
	secondSink := &lifecycleSink{flushErr: secondErr}
	logger := NewLogger(LoggerConfig{
		Sinks: []Sink{firstSink, plainSink, secondSink},
	})

	err := logger.Flush()
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("Flush() error = %v, want both sink errors", err)
	}
	if !strings.Contains(err.Error(), "flush sink 0") || !strings.Contains(err.Error(), "flush sink 2") {
		t.Fatalf("Flush() error = %q, want sink indexes", err)
	}
	if firstSink.flushCalls != 1 || secondSink.flushCalls != 1 {
		t.Fatalf("flush calls = (%d, %d), want (1, 1)", firstSink.flushCalls, secondSink.flushCalls)
	}
}

func TestLoggerClosesEverySupportedSinkAndJoinsErrors(t *testing.T) {
	firstErr := errors.New("first close failed")
	secondErr := errors.New("second close failed")
	firstSink := &lifecycleSink{closeErr: firstErr}
	plainSink := &recordingSink{}
	secondSink := &lifecycleSink{closeErr: secondErr}
	logger := NewLogger(LoggerConfig{
		Sinks: []Sink{firstSink, plainSink, secondSink},
	})

	err := logger.Close()
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("Close() error = %v, want both sink errors", err)
	}
	if !strings.Contains(err.Error(), "close sink 0") || !strings.Contains(err.Error(), "close sink 2") {
		t.Fatalf("Close() error = %q, want sink indexes", err)
	}
	if firstSink.closeCalls != 1 || secondSink.closeCalls != 1 {
		t.Fatalf("close calls = (%d, %d), want (1, 1)", firstSink.closeCalls, secondSink.closeCalls)
	}
}

func TestLoggerLifecycleSucceedsWithoutSupportedSinks(t *testing.T) {
	logger := NewLogger(LoggerConfig{Sinks: []Sink{&recordingSink{}}})

	if err := logger.Flush(); err != nil {
		t.Fatalf("Flush() error = %v, want nil", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

func TestLoggerFatalLogsClosesSinksAndExits(t *testing.T) {
	sink := &fatalSink{}
	logger := NewLogger(LoggerConfig{
		MinLevel:  InfoLevel,
		Formatter: &recordingFormatter{},
		Sinks:     []Sink{sink},
	})
	exited := make(chan int, 1)
	logger.exit = func(code int) {
		exited <- code
	}

	logger.Fatal("fatal message %d", 7)

	if got := <-exited; got != 1 {
		t.Fatalf("exit code = %d, want 1", got)
	}
	if sink.message != "formatted record" {
		t.Fatalf("sink message = %q, want formatted record", sink.message)
	}
	if sink.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", sink.closeCalls)
	}
}

type recordingFormatter struct {
	record Record
	calls  int
}

func (f *recordingFormatter) Format(record Record) string {
	f.record = record
	f.calls++
	return "formatted record"
}

type recordingSink struct {
	messages []string
}

func (s *recordingSink) Log(message string) {
	s.messages = append(s.messages, message)
}

type lifecycleSink struct {
	flushCalls int
	closeCalls int
	flushErr   error
	closeErr   error
}

func (s *lifecycleSink) Log(string) {}

func (s *lifecycleSink) Flush() error {
	s.flushCalls++
	return s.flushErr
}

func (s *lifecycleSink) Close() error {
	s.closeCalls++
	return s.closeErr
}

type fatalSink struct {
	message    string
	closeCalls int
}

func (s *fatalSink) Log(message string) {
	s.message = message
}

func (s *fatalSink) Close() error {
	s.closeCalls++
	return nil
}
