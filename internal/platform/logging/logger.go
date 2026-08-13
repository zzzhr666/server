package logging

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Logger struct {
	minLevel  Level
	formatter Formatter
	sinks     []Sink
	exit      func(int)
}

type LoggerConfig struct {
	MinLevel  Level
	Formatter Formatter
	Sinks     []Sink
}

func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		MinLevel:  InfoLevel,
		Formatter: NewDefaultFormatter(DefaultPattern),
		Sinks:     []Sink{},
	}
}

func NewLogger(config LoggerConfig) *Logger {
	return &Logger{
		minLevel:  config.MinLevel,
		formatter: config.Formatter,
		sinks:     config.Sinks,
		exit:      os.Exit,
	}
}

func (l *Logger) Trace(fmt string, args ...any) {
	l.log(TraceLevel, fmt, args...)
}

func (l *Logger) Debug(fmt string, args ...any) {
	l.log(DebugLevel, fmt, args...)
}

func (l *Logger) Info(fmt string, args ...any) {
	l.log(InfoLevel, fmt, args...)
}

func (l *Logger) Warn(fmt string, args ...any) {
	l.log(WarnLevel, fmt, args...)
}

func (l *Logger) Error(fmt string, args ...any) {
	l.log(ErrorLevel, fmt, args...)
}

// Fatal 记录致命错误，关闭所有 Sink，并以非零状态结束进程。
func (l *Logger) Fatal(fmt string, args ...any) {
	l.log(FatalLevel, fmt, args...)
	_ = l.Close()
	l.exit(1)
}

func (l *Logger) shouldLog(level Level) bool {
	return level >= l.minLevel
}

func (l *Logger) log(level Level, format string, args ...any) {
	if !l.shouldLog(level) {
		return
	}
	location := captureSourceLocation()
	record := Record{
		Time:     time.Now(),
		Level:    level,
		Message:  fmt.Sprintf(format, args...),
		File:     location.File,
		Function: location.Function,
		Line:     location.Line,
	}
	message := l.formatter.Format(record)
	for _, sink := range l.sinks {
		sink.Log(message)
	}
}

// Flush 刷新所有支持显式刷新的 Sink。
func (l *Logger) Flush() error {
	errs := make([]error, 0)
	for index, sink := range l.sinks {
		flusher, ok := sink.(Flusher)
		if ok {
			if err := flusher.Flush(); err != nil {
				errs = append(errs, fmt.Errorf("logging: flush sink %d: %w", index, err))
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Close 关闭所有持有资源的 Sink。
func (l *Logger) Close() error {
	errs := make([]error, 0)
	for index, sink := range l.sinks {
		closer, ok := sink.(Closer)
		if ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("logging: close sink %d: %w", index, err))
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
