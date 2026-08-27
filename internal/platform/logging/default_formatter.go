package logging

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DefaultFormatter struct {
	pattern string
	parts   []formatPart
}

const DefaultPattern = "[%Y/%m/%d][%H:%M:%S.%e][%l][%s:%!:%#] %v"

// NewDefaultFormatter 使用指定模式创建默认日志格式化器。
func NewDefaultFormatter(pattern string) *DefaultFormatter {
	return &DefaultFormatter{
		pattern: pattern,
		parts:   parsePattern(pattern),
	}
}

// Format 按配置模式将日志记录格式化为文本。
func (d *DefaultFormatter) Format(record Record) string {
	buffer := make([]byte, 0, len(d.pattern)+len(record.Message)+64)
	for _, part := range d.parts {
		switch part.kind {
		case partLiteral:
			buffer = append(buffer, part.text...)
		case partYear:
			buffer = record.Time.AppendFormat(buffer, "2006")
		case partMonth:
			buffer = record.Time.AppendFormat(buffer, "01")
		case partDay:
			buffer = record.Time.AppendFormat(buffer, "02")
		case partHour:
			buffer = record.Time.AppendFormat(buffer, "15")
		case partMinute:
			buffer = record.Time.AppendFormat(buffer, "04")
		case partSecond:
			buffer = record.Time.AppendFormat(buffer, "05")
		case partMillisecond:
			buffer = appendMilliseconds(buffer, record.Time)
		case partLevel:
			buffer = append(buffer, record.Level.String()...)
		case partFile:
			buffer = append(buffer, filepath.Base(record.File)...)
		case partFunction:
			buffer = append(buffer, shortFunctionName(record.Function)...)
		case partLine:
			buffer = strconv.AppendInt(buffer, int64(record.Line), 10)
		case partMessage:
			buffer = append(buffer, record.Message...)
		}
	}
	return string(buffer)
}

func shortFunctionName(function string) string {
	if index := strings.LastIndex(function, "/"); index >= 0 {
		function = function[index+1:]
	}
	return function
}

func appendMilliseconds(buffer []byte, value time.Time) []byte {
	milliseconds := value.Nanosecond() / int(time.Millisecond)
	if milliseconds < 100 {
		buffer = append(buffer, '0')
	}
	if milliseconds < 10 {
		buffer = append(buffer, '0')
	}
	return strconv.AppendInt(buffer, int64(milliseconds), 10)
}
