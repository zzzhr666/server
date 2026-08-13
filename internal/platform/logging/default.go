package logging

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

var defaultState struct {
	mu         sync.Mutex
	once       sync.Once
	configured bool
	started    bool
	options    DefaultLoggerOptions
	logger     *Logger
	err        error
}

// ConfigureDefault 设置进程级默认 Logger 的懒加载配置。
// 必须在第一次调用 Default 或包级日志方法前调用。
func ConfigureDefault(options DefaultLoggerOptions) error {
	defaultState.mu.Lock()
	defer defaultState.mu.Unlock()
	if defaultState.configured || defaultState.started {
		return errors.New("logging: default logger has already been configured")
	}
	defaultState.options = options
	defaultState.configured = true
	return nil
}

// Default 返回进程级默认 Logger；首次调用时才创建底层 Sink。
func Default() (*Logger, error) {
	defaultState.mu.Lock()
	defaultState.started = true
	options := defaultState.options
	defaultState.mu.Unlock()

	defaultState.once.Do(func() {
		defaultState.logger, defaultState.err = NewDefaultLogger(options)
	})
	return defaultState.logger, defaultState.err
}

// CloseDefault 关闭进程级默认 Logger。
func CloseDefault() error {
	logger, err := Default()
	if err != nil {
		return err
	}
	return logger.Close()
}

// Trace 使用进程级默认 Logger 记录 Trace 日志。
func Trace(format string, args ...any) {
	logger, err := Default()
	if err != nil {
		reportDefaultError(err)
		return
	}
	logger.log(TraceLevel, format, args...)
}

// Debug 使用进程级默认 Logger 记录 Debug 日志。
func Debug(format string, args ...any) {
	logger, err := Default()
	if err != nil {
		reportDefaultError(err)
		return
	}
	logger.log(DebugLevel, format, args...)
}

// Info 使用进程级默认 Logger 记录 Info 日志。
func Info(format string, args ...any) {
	logger, err := Default()
	if err != nil {
		reportDefaultError(err)
		return
	}
	logger.log(InfoLevel, format, args...)
}

// Warn 使用进程级默认 Logger 记录 Warn 日志。
func Warn(format string, args ...any) {
	logger, err := Default()
	if err != nil {
		reportDefaultError(err)
		return
	}
	logger.log(WarnLevel, format, args...)
}

// Error 使用进程级默认 Logger 记录 Error 日志。
func Error(format string, args ...any) {
	logger, err := Default()
	if err != nil {
		reportDefaultError(err)
		return
	}
	logger.log(ErrorLevel, format, args...)
}

// Fatal 记录 Fatal 日志，关闭进程级默认 Logger，并结束进程。
func Fatal(format string, args ...any) {
	logger, err := Default()
	if err != nil {
		reportDefaultError(err)
		os.Exit(1)
	}
	logger.log(FatalLevel, format, args...)
	_ = logger.Close()
	logger.exit(1)
}

func reportDefaultError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "logging: initialize default logger: %v\n", err)
}
