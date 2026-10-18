package main

import "testing"

func TestMaskValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"short", "short"},           // <= 8 chars: shown as-is
		{"exactly8", "exactly8"},     // 8 chars: as-is
		{"sk-abcdefgh", "sk-abc…"},   // > 8: first 6 runes + …
		{"sk-4b42abcdef", "sk-4b4…"}, // README example shape
		{"密钥值很长很长很长", "密钥值很长很…"},     // multi-byte safe (10 runes)
	}
	for _, tt := range tests {
		if got := maskValue(tt.in); got != tt.want {
			t.Errorf("maskValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
