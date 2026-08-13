package logging

import "testing"

func TestParseMode(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  Mode
	}{
		{name: "debug", value: "debug", want: DebugMode},
		{name: "release", value: " RELEASE ", want: ReleaseMode},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseMode(test.value)
			if err != nil {
				t.Fatalf("ParseMode() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ParseMode() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestParseModeRejectsUnsupportedValue(t *testing.T) {
	if _, err := ParseMode("relase"); err == nil {
		t.Fatal("ParseMode() error = nil, want an error")
	}
}
