package secrets

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{"basic", "a=1\nb=2\n", map[string]string{"a": "1", "b": "2"}},
		{"comments and blanks", "# comment\n\n  \na=1\n", map[string]string{"a": "1"}},
		{"value contains equals", "jwt=eyJ.abc==\n", map[string]string{"jwt": "eyJ.abc=="}},
		{"whitespace trimmed around", "  a  =  1  \n", map[string]string{"a": "1"}},
		{"inner whitespace kept", "a=hello world\n", map[string]string{"a": "hello world"}},
		{"hash inside value kept", "a=value#note\n", map[string]string{"a": "value#note"}},
		{"crlf endings", "a=1\r\nb=2\r\n", map[string]string{"a": "1", "b": "2"}},
		{"duplicate last wins", "a=1\na=2\n", map[string]string{"a": "2"}},
		{"empty key ignored", "=1\na=2\n", map[string]string{"a": "2"}},
		{"empty value allowed", "a=\n", map[string]string{"a": ""}},
		{"line without equals ignored", "a=1\nbroken\nb=2\n", map[string]string{"a": "1", "b": "2"}},
		{"no trailing newline", "a=1", map[string]string{"a": "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.secrets")
	if err := os.WriteFile(path, []byte("# comment\nkey=value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	want := map[string]string{"key": "value"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %v, want %v", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("Load() expected error for missing file")
	}
}
