package player

import (
	"errors"
	"testing"
)

func TestResolveAvatar(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    AvatarType
		wantErr error
	}{
		{name: "empty uses default", value: "", want: DefaultAvatar},
		{name: "whitespace uses default", value: "   ", want: DefaultAvatar},
		{name: "valid avatar", value: "mage", want: AvatarMage},
		{name: "trims valid avatar", value: " mage ", want: AvatarMage},
		{name: "does not normalize case", value: "Mage", wantErr: ErrInvalidAvatar},
		{name: "rejects URL", value: "https://example.com/avatar.png", wantErr: ErrInvalidAvatar},
		{name: "rejects unknown avatar", value: "unknown", wantErr: ErrInvalidAvatar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAvatar(tt.value)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveAvatar(%q) error = %v, want %v", tt.value, err, tt.wantErr)
			}
			if tt.wantErr == nil && got != tt.want {
				t.Fatalf("ResolveAvatar(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
