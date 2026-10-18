package main

import (
	"os"
	"strings"
	"testing"
)

func TestReadPipedValue(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"single line", "sk-abc", "sk-abc", false},
		{"trailing newline", "sk-abc\n", "sk-abc", false},
		{"crlf", "sk-abc\r\n", "sk-abc", false},
		{"surrounding spaces trimmed", "  sk-abc \n", "sk-abc", false},
		{"empty", "", "", true},
		{"only newline", "\n", "", true},
		{"two lines", "a\nb\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readPipedValue(strings.NewReader(tt.in))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("readPipedValue(%q) = %q, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("readPipedValue(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("readPipedValue(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsTerminalRegularFileIsFalse(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("a regular file must not count as a terminal")
	}
}
