package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 14, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))

func parseString(t *testing.T, s string) *File {
	t.Helper()
	f, err := ParseFile(strings.NewReader(s))
	if err != nil {
		t.Fatalf("ParseFile() error: %v", err)
	}
	return f
}

func TestParseMetadata(t *testing.T) {
	src := `# a comment
openai_api_key=sk-abc
#@meta openai_api_key created=2026-09-14T10:00:00+08:00 modified=2026-09-14T12:00:00+08:00
#@note openai_api_key base_url=https://api.deepseek.com
#@note openai_api_key account=kwelldo

github_password=hunter2
#@note github_password account=kwelldo
#@note github_password url=https://github.com/login with a trailing note
`
	f := parseString(t, src)
	if got := f.Names(); len(got) != 2 || got[0] != "openai_api_key" || got[1] != "github_password" {
		t.Fatalf("Names() = %v", got)
	}

	e := f.Entry("openai_api_key")
	if e == nil {
		t.Fatal("openai_api_key missing")
	}
	if e.Value != "sk-abc" {
		t.Errorf("value = %q", e.Value)
	}
	want, _ := time.Parse(time.RFC3339, "2026-09-14T10:00:00+08:00")
	if !e.Created.Equal(want) {
		t.Errorf("created = %v, want %v", e.Created, want)
	}
	if v, ok := e.Note("base_url"); !ok || v != "https://api.deepseek.com" {
		t.Errorf("note base_url = %q, %v", v, ok)
	}
	if len(e.Notes) != 2 || e.Notes[0].Label != "base_url" || e.Notes[1].Label != "account" {
		t.Errorf("notes = %+v", e.Notes)
	}

	// Values with spaces survive; the label is what precedes the first '='.
	g := f.Entry("github_password")
	if v, _ := g.Note("url"); v != "https://github.com/login with a trailing note" {
		t.Errorf("note url = %q", v)
	}
	if !g.Created.IsZero() || !g.Modified.IsZero() {
		t.Errorf("github_password should have unknown dates, got %v/%v", g.Created, g.Modified)
	}
}

// Metadata is only recognised on lines starting with the exact prefixes, so
// entry values that happen to look like metadata stay untouched.
func TestParseIgnoresLookalikes(t *testing.T) {
	f := parseString(t, "a=#@meta a created=2026-09-14T10:00:00+08:00\n#@meta notakey\n#@note\n")
	e := f.Entry("a")
	if e == nil || !strings.HasPrefix(e.Value, "#@meta") {
		t.Fatalf("value = %+v", e)
	}
	if !e.Created.IsZero() {
		t.Errorf("created should be unknown, got %v", e.Created)
	}
}

func TestSetStampsTimestamps(t *testing.T) {
	f := parseString(t, "a=1\n#@meta a created=2020-01-01T00:00:00+08:00\n")
	later := t0.Add(time.Hour)
	if _, err := f.Set("a", "2", later); err != nil {
		t.Fatal(err)
	}
	e := f.Entry("a")
	if e.Value != "2" {
		t.Errorf("value = %q", e.Value)
	}
	if !e.Modified.Equal(later) {
		t.Errorf("modified = %v, want %v", e.Modified, later)
	}
	// Created is preserved once known.
	want, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00+08:00")
	if !e.Created.Equal(want) {
		t.Errorf("created = %v, want %v", e.Created, want)
	}
}

func TestSetPreservesCommentsAndOrder(t *testing.T) {
	src := "# migrated from ai-key.txt\na=1\nb=2\n# 2026-06-30 新增\nc=3\n"
	f := parseString(t, src)
	if _, err := f.Set("b", "22", t0); err != nil {
		t.Fatal(err)
	}
	got := string(f.Bytes())
	want := "# migrated from ai-key.txt\na=1\nb=22\n#@meta b created=2026-09-14T12:00:00+08:00 modified=2026-09-14T12:00:00+08:00\n# 2026-06-30 新增\nc=3\n"
	if got != want {
		t.Errorf("Bytes() =\n%q\nwant\n%q", got, want)
	}
}

func TestSetNewKeyAppendsAtEnd(t *testing.T) {
	f := parseString(t, "a=1\n# trailing comment\n")
	if _, err := f.Set("b", "2", t0); err != nil {
		t.Fatal(err)
	}
	want := "a=1\n# trailing comment\nb=2\n#@meta b created=2026-09-14T12:00:00+08:00 modified=2026-09-14T12:00:00+08:00\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("Bytes() =\n%q\nwant\n%q", got, want)
	}
}

// Duplicate names collapse on render: the last (winning) line keeps its
// position and the superseded ones disappear.
func TestDuplicatesCollapseOnRender(t *testing.T) {
	f := parseString(t, "a=1\nb=2\na=3\n")
	if v := f.Entry("a").Value; v != "3" {
		t.Errorf("last should win, value = %q", v)
	}
	want := "b=2\na=3\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("Bytes() = %q, want %q", got, want)
	}
}

func TestDeleteRemovesEntryAndMetadata(t *testing.T) {
	src := "a=1\n#@note a base_url=https://example.com\n# keep me\nb=2\n#@meta b created=2026-09-14T10:00:00+08:00\n"
	f := parseString(t, src)
	if !f.Delete("a") {
		t.Fatal("Delete(a) = false")
	}
	if f.Delete("a") {
		t.Error("second Delete(a) should report false")
	}
	want := "# keep me\nb=2\n#@meta b created=2026-09-14T10:00:00+08:00\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("Bytes() =\n%q\nwant\n%q", got, want)
	}
}

func TestOrphanMetadataIsPreserved(t *testing.T) {
	src := "#@note ghost base_url=https://example.com\na=1\n"
	f := parseString(t, src)
	if got := string(f.Bytes()); got != src {
		t.Errorf("Bytes() = %q, want %q", got, src)
	}
}

func TestNotes(t *testing.T) {
	f := parseString(t, "a=1\n")
	if err := f.SetNote("a", "base_url", "https://x.example", t0); err != nil {
		t.Fatal(err)
	}
	if err := f.SetNote("a", "account", "kwelldo", t0); err != nil {
		t.Fatal(err)
	}
	// Replacing keeps the original position.
	if err := f.SetNote("a", "base_url", "https://y.example", t0); err != nil {
		t.Fatal(err)
	}
	e := f.Entry("a")
	if len(e.Notes) != 2 || e.Notes[0].Label != "base_url" || e.Notes[0].Value != "https://y.example" {
		t.Fatalf("notes = %+v", e.Notes)
	}
	if !e.Modified.Equal(t0) {
		t.Errorf("modified = %v, want %v", e.Modified, t0)
	}

	want := "a=1\n#@meta a created=2026-09-14T12:00:00+08:00 modified=2026-09-14T12:00:00+08:00\n" +
		"#@note a base_url=https://y.example\n#@note a account=kwelldo\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("Bytes() =\n%q\nwant\n%q", got, want)
	}

	removed, err := f.DeleteNote("a", "account", t0)
	if err != nil || !removed {
		t.Fatalf("DeleteNote() = %v, %v", removed, err)
	}
	if removed, _ := f.DeleteNote("a", "account", t0); removed {
		t.Error("second DeleteNote should report false")
	}
	if err := f.SetNote("nope", "x", "y", t0); err == nil {
		t.Error("SetNote on a missing entry should fail")
	}
}

func TestBackfill(t *testing.T) {
	f := parseString(t, "a=1\nb=2\n#@meta b created=2020-01-01T00:00:00+08:00 modified=2020-01-01T00:00:00+08:00\n")
	if n := f.Backfill(t0, t0); n != 1 {
		t.Fatalf("Backfill() = %d, want 1", n)
	}
	if n := f.Backfill(t0, t0); n != 0 {
		t.Errorf("second Backfill() = %d, want 0", n)
	}
	if !f.Entry("a").Created.Equal(t0) {
		t.Errorf("a.created = %v", f.Entry("a").Created)
	}
	// b keeps its dates.
	want, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00+08:00")
	if !f.Entry("b").Created.Equal(want) {
		t.Errorf("b.created = %v, want %v", f.Entry("b").Created, want)
	}
}

func TestValidation(t *testing.T) {
	f := parseString(t, "a=1\n")
	for _, name := range []string{"", "has space", "has\ttab", "has#hash", "has=equals", "new\nline"} {
		if _, err := f.Set(name, "v", t0); err == nil {
			t.Errorf("Set(%q) should fail", name)
		}
	}
	if _, err := f.Set("k", "line\nbreak", t0); err == nil {
		t.Error("Set with a newline in the value should fail")
	}
	if err := f.SetNote("a", "bad=label", "v", t0); err == nil {
		t.Error("SetNote with '=' in the label should fail")
	}
}

// Render → parse → render must be stable, and the flat map must not change.
func TestRoundTripIsStable(t *testing.T) {
	src := `# migrated
a=1
b=2
#@meta b created=2026-09-14T10:00:00+08:00 modified=2026-09-14T11:00:00+08:00
#@note b base_url=https://example.com/path?a=b=c
jwt=eyJ.abc==
empty=
# trailing comment
`
	f := parseString(t, src)
	once := string(f.Bytes())
	twice := string(parseString(t, once).Bytes())
	if once != twice {
		t.Errorf("not stable:\n%q\n%q", once, twice)
	}
	if once != src {
		t.Errorf("normalisation lost data:\n%q\nwant\n%q", once, src)
	}

	m, err := Parse(strings.NewReader(once))
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 4 || m["jwt"] != "eyJ.abc==" || m["empty"] != "" {
		t.Errorf("flat parse after round trip = %v", m)
	}
}

func TestLoadFileAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".secrets")
	if err := os.WriteFile(path, []byte("a=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Path() != path {
		t.Errorf("Path() = %q", f.Path())
	}
	if _, err := f.Set("b", "2", t0); err != nil {
		t.Fatal(err)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := st.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
	if _, err := os.Stat(filepath.Join(dir, ".secrets-1.tmp")); !os.IsNotExist(err) {
		t.Errorf("temp file left behind: %v", err)
	}

	again, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := again.Entry("b").Value; v != "2" {
		t.Errorf("b = %q", v)
	}
	if v := again.Entry("a").Value; v != "1" {
		t.Errorf("a = %q", v)
	}
}

func TestLoadFileMissing(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsNotExist(err) {
		t.Errorf("IsNotExist(%v) = false", err)
	}
}

func TestNewFileCreatesFromScratch(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".secrets")
	f := NewFile(path)
	if _, err := f.Set("a", "1", t0); err != nil {
		t.Fatal(err)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := again.Entry("a").Value; v != "1" {
		t.Errorf("a = %q", v)
	}
}
