package logging

import (
	"fmt"
	"strings"
)

type Mode uint8

const (
	DebugMode Mode = iota
	ReleaseMode
)

func (m Mode) String() string {
	switch m {
	case DebugMode:
		return "debug"
	case ReleaseMode:
		return "release"
	default:
		return "unknown"
	}
}

// ParseMode 将日志运行模式字符串转换为 Mode。
func ParseMode(value string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return DebugMode, nil
	case "release":
		return ReleaseMode, nil
	default:
		return DebugMode, fmt.Errorf("logging: unsupported mode %q", value)
	}
}
