package logging

import (
	"reflect"
	"testing"
)

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []formatPart
	}{
		{
			name:    "empty pattern",
			pattern: "",
			want:    nil,
		},
		{
			name:    "literal only",
			pattern: "plain text",
			want:    []formatPart{{kind: partLiteral, text: "plain text"}},
		},
		{
			name:    "adjacent placeholders",
			pattern: "%Y%m%d",
			want: []formatPart{
				{kind: partYear},
				{kind: partMonth},
				{kind: partDay},
			},
		},
		{
			name:    "escaped percent",
			pattern: "progress=100%%",
			want: []formatPart{
				{kind: partLiteral, text: "progress=100"},
				{kind: partLiteral, text: "%"},
			},
		},
		{
			name:    "unsupported placeholder remains literal",
			pattern: "[%x] %v",
			want: []formatPart{
				{kind: partLiteral, text: "["},
				{kind: partLiteral, text: "%x"},
				{kind: partLiteral, text: "] "},
				{kind: partMessage},
			},
		},
		{
			name:    "trailing percent remains literal",
			pattern: "message %",
			want: []formatPart{
				{kind: partLiteral, text: "message "},
				{kind: partLiteral, text: "%"},
			},
		},
		{
			name:    "unicode literal",
			pattern: "日志[%l] %v",
			want: []formatPart{
				{kind: partLiteral, text: "日志["},
				{kind: partLevel},
				{kind: partLiteral, text: "] "},
				{kind: partMessage},
			},
		},
		{
			name:    "all supported placeholders",
			pattern: "%Y%m%d%H%M%S%e%l%s%!%#%v",
			want: []formatPart{
				{kind: partYear},
				{kind: partMonth},
				{kind: partDay},
				{kind: partHour},
				{kind: partMinute},
				{kind: partSecond},
				{kind: partMillisecond},
				{kind: partLevel},
				{kind: partFile},
				{kind: partFunction},
				{kind: partLine},
				{kind: partMessage},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parsePattern(test.pattern); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parsePattern(%q) = %#v, want %#v", test.pattern, got, test.want)
			}
		})
	}
}
