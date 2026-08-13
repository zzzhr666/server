package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultFormatterFormatsDefaultPattern(t *testing.T) {
	formatter := NewDefaultFormatter(DefaultPattern)
	record := testRecord(t, "2026-08-13T10:24:35.123+08:00")

	got := formatter.Format(record)
	want := "[2026/08/13][10:24:35.123][info][main.go:logic-server.main:42] server started"
	if got != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestDefaultFormatterFormatsCustomPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "message only", pattern: "%v", want: "server started"},
		{name: "level and source", pattern: "[%l][%s:%#] %v", want: "[info][main.go:42] server started"},
		{name: "compact time", pattern: "%Y%m%d-%H%M%S.%e", want: "20260813-102435.123"},
		{name: "escaped percent", pattern: "100%% %v", want: "100% server started"},
		{name: "unsupported placeholder", pattern: "[%x] %v", want: "[%x] server started"},
		{name: "trailing percent", pattern: "%v %", want: "server started %"},
		{name: "unicode literal", pattern: "日志[%l] %v", want: "日志[info] server started"},
		{name: "empty pattern", pattern: "", want: ""},
	}

	record := testRecord(t, "2026-08-13T10:24:35.123+08:00")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NewDefaultFormatter(test.pattern).Format(record); got != test.want {
				t.Fatalf("Format() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDefaultFormatterPadsMilliseconds(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		want      string
	}{
		{name: "single digit", timestamp: "2026-08-13T10:24:35.003+08:00", want: "003"},
		{name: "two digits", timestamp: "2026-08-13T10:24:35.042+08:00", want: "042"},
		{name: "three digits", timestamp: "2026-08-13T10:24:35.123+08:00", want: "123"},
	}

	formatter := NewDefaultFormatter("%e")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatter.Format(testRecord(t, test.timestamp)); got != test.want {
				t.Fatalf("Format() = %q, want %q", got, test.want)
			}
		})
	}
}

func testRecord(t *testing.T, timestamp string) Record {
	t.Helper()
	return Record{
		Time:     mustParseTime(t, timestamp),
		Level:    InfoLevel,
		Message:  "server started",
		File:     filepath.Join("home", "server", "cmd", "logic-server", "main.go"),
		Function: "server/cmd/logic-server.main",
		Line:     42,
	}
}
