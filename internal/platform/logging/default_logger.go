package logging

import (
	"errors"
	"fmt"
	"path/filepath"
)

// DefaultLoggerOptions 定义默认 Logger 的等级、运行模式和文件命名信息。
type DefaultLoggerOptions struct {
	Level       Level
	Mode        Mode
	ServiceName string
	LogDir      string
}

// NewDefaultLogger 按运行模式创建默认 Formatter 和日志 Sink。
// DebugMode 使用 ConsoleSink 与 AsyncFileSink，ReleaseMode 只使用 AsyncFileSink。
func NewDefaultLogger(options DefaultLoggerOptions) (*Logger, error) {
	if options.ServiceName == "" {
		return nil, errors.New("logging: service name is empty")
	}
	if options.LogDir == "" {
		options.LogDir = "./logs"
	}
	if options.Mode != DebugMode && options.Mode != ReleaseMode {
		return nil, fmt.Errorf("logging: unsupported mode %d", options.Mode)
	}

	fileSink, err := NewAsyncFileSink(FileSinkConfig{
		Path: filepath.Join(options.LogDir, options.ServiceName+".log"),
	})
	if err != nil {
		return nil, fmt.Errorf("logging: create file sink: %w", err)
	}

	sinks := make([]Sink, 0, 2)
	if options.Mode == DebugMode {
		sinks = append(sinks, &ConsoleSink{})
	}
	sinks = append(sinks, fileSink)

	return NewLogger(LoggerConfig{
		MinLevel:  options.Level,
		Formatter: NewDefaultFormatter(DefaultPattern),
		Sinks:     sinks,
	}), nil
}
