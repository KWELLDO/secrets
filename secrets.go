// Package secrets implements a minimal flat key-value secrets file format
// with optional per-key metadata (timestamps and notes).
//
// Format (see ~/Documents/secrets-format-spec.md):
//
//   - one entry per line: name=value
//   - split on the FIRST '='; the rest of the line is the value (may contain '=')
//   - whitespace around name and value is trimmed; inner whitespace is kept
//   - empty lines are ignored
//   - lines starting with '#' are comments and ignored
//   - a line with an empty name is ignored
//   - duplicate names: the last occurrence wins
//   - empty values are allowed (name=)
//   - CRLF line endings are accepted (trailing '\r' is stripped)
//   - no quoting, no escaping: values are taken literally
//
// Metadata is stored in comment lines, so every existing flat parser keeps
// working untouched:
//
//	#@meta <name> created=<RFC3339> modified=<RFC3339>
//	#@note <name> <label>=<value>
//
// Both lines are optional, may repeat, and are attached to <name> explicitly.
// A "#@note" value may contain spaces and '='; the label is everything before
// the first '=' and may contain spaces too, but not '='.
package secrets

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Prefixes of the metadata comment lines.
const (
	MetaPrefix = "#@meta "
	NotePrefix = "#@note "
)

// Note is one labelled annotation attached to an entry. Labels are unique
// within an entry; setting an existing label overwrites its value in place.
type Note struct {
	Label string
	Value string
}

// Entry is a single secret: its name, value, timestamps and notes.
// A zero Created/Modified means "unknown" (the entry predates metadata).
type Entry struct {
	Name     string
	Value    string
	Created  time.Time
	Modified time.Time
	Notes    []Note
}

// Note returns the value stored under label.
func (e *Entry) Note(label string) (string, bool) {
	for _, n := range e.Notes {
		if n.Label == label {
			return n.Value, true
		}
	}
	return "", false
}

// ValidateName rejects names that cannot be represented in the file format.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("empty name")
	}
	if strings.ContainsAny(name, " \t\r\n#=") {
		return fmt.Errorf("invalid name %q (no whitespace, '#' or '=')", name)
	}
	return nil
}

func validateNoteLabel(label string) error {
	if label == "" {
		return errors.New("empty note label")
	}
	if strings.ContainsAny(label, "=\r\n") {
		return fmt.Errorf("invalid note label %q (no '=')", label)
	}
	return nil
}

func validateLineValue(what, v string) error {
	if strings.ContainsAny(v, "\r\n") {
		return fmt.Errorf("%s must not contain a newline", what)
	}
	return nil
}

// ---------------------------------------------------------------------------
// parsing
// ---------------------------------------------------------------------------

type lineKind uint8

const (
	lineOther lineKind = iota // blank, plain comment, or anything unrecognised
	lineEntry
	lineMeta
	lineNote
)

type lineInfo struct {
	kind lineKind
	name string
}

// File is an in-memory view of a secrets file. It remembers the original
// lines so that comments, blank lines and ordering survive a rewrite.
type File struct {
	path          string
	lines         []string
	infos         []lineInfo
	entries       []*Entry
	index         map[string]*Entry
	lastEntryLine map[string]int
	deleted       map[string]bool
}

// NewFile returns an empty file bound to path.
func NewFile(path string) *File {
	return &File{
		path:          path,
		index:         make(map[string]*Entry),
		lastEntryLine: make(map[string]int),
		deleted:       make(map[string]bool),
	}
}

// Path returns the path the file was loaded from ("" if parsed from a reader).
func (f *File) Path() string { return f.path }

// Entries returns the entries in file order.
func (f *File) Entries() []*Entry { return f.entries }

// Entry returns the named entry, or nil.
func (f *File) Entry(name string) *Entry { return f.index[name] }

// Names returns the entry names in file order.
func (f *File) Names() []string {
	names := make([]string, 0, len(f.entries))
	for _, e := range f.entries {
		names = append(names, e.Name)
	}
	return names
}

// SortedNames returns the entry names sorted lexicographically.
func (f *File) SortedNames() []string {
	names := f.Names()
	sort.Strings(names)
	return names
}

// SortedEntries returns the entries sorted by name.
func (f *File) SortedEntries() []*Entry {
	out := make([]*Entry, len(f.entries))
	copy(out, f.entries)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// LoadFile reads and parses the file at path.
func LoadFile(path string) (*File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	f, err := ParseFile(fh)
	if err != nil {
		return nil, err
	}
	f.path = path
	return f, nil
}

// ParseFile parses entries and metadata from r.
func ParseFile(r io.Reader) (*File, error) {
	f := NewFile("")
	created := make(map[string]time.Time)
	modified := make(map[string]time.Time)
	notes := make(map[string][]Note)

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		i := len(f.lines)
		f.lines = append(f.lines, line)

		info := lineInfo{kind: lineOther}
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), MetaPrefix):
			name, c, m, ok := parseMetaLine(line)
			if ok {
				info = lineInfo{kind: lineMeta, name: name}
				if !c.IsZero() {
					created[name] = c
				}
				if !m.IsZero() {
					modified[name] = m
				}
			}
		case strings.HasPrefix(strings.TrimSpace(line), NotePrefix):
			name, label, value, ok := parseNoteLine(line)
			if ok {
				info = lineInfo{kind: lineNote, name: name}
				notes[name] = setNote(notes[name], label, value)
			}
		default:
			if name, value, ok := parseEntryLine(line); ok {
				info = lineInfo{kind: lineEntry, name: name}
				f.lastEntryLine[name] = i
				if e, exists := f.index[name]; exists {
					e.Value = value // duplicate: last wins
				} else {
					e := &Entry{Name: name, Value: value}
					f.index[name] = e
					f.entries = append(f.entries, e)
				}
			}
		}
		f.infos = append(f.infos, info)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	for _, e := range f.entries {
		e.Created = created[e.Name]
		e.Modified = modified[e.Name]
		e.Notes = notes[e.Name]
	}
	return f, nil
}

// parseEntryLine splits "name=value". It returns false for blank lines,
// comments and lines without '=' (those are preserved verbatim).
func parseEntryLine(line string) (name, value string, ok bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") {
		return "", "", false
	}
	k, v, found := strings.Cut(t, "=")
	if !found {
		return "", "", false
	}
	k = strings.TrimSpace(k)
	if k == "" {
		return "", "", false
	}
	return k, strings.TrimSpace(v), true
}

// parseMetaLine parses "#@meta <name> created=<t> modified=<t>".
func parseMetaLine(line string) (name string, created, modified time.Time, ok bool) {
	rest := strings.TrimSpace(line)[len(MetaPrefix):]
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return "", time.Time{}, time.Time{}, false
	}
	name = fields[0]
	for _, fld := range fields[1:] {
		k, v, found := strings.Cut(fld, "=")
		if !found {
			continue
		}
		switch k {
		case "created":
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				created = t
			}
		case "modified":
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				modified = t
			}
		}
	}
	return name, created, modified, true
}

// parseNoteLine parses "#@note <name> <label>=<value>". The value may contain
// spaces and '='.
func parseNoteLine(line string) (name, label, value string, ok bool) {
	rest := strings.TrimSpace(line)[len(NotePrefix):]
	i := strings.IndexAny(rest, " \t")
	if i < 0 {
		return "", "", "", false
	}
	name = rest[:i]
	rest = strings.TrimSpace(rest[i+1:])
	label, value, found := strings.Cut(rest, "=")
	label = strings.TrimSpace(label)
	if !found || name == "" || label == "" {
		return "", "", "", false
	}
	return name, label, value, true
}

func setNote(list []Note, label, value string) []Note {
	for i := range list {
		if list[i].Label == label {
			list[i].Value = value
			return list
		}
	}
	return append(list, Note{Label: label, Value: value})
}

func deleteNote(list []Note, label string) ([]Note, bool) {
	for i := range list {
		if list[i].Label == label {
			return append(list[:i:i], list[i+1:]...), true
		}
	}
	return list, false
}

// Parse reads name=value entries from r and returns them as a map, ignoring
// all metadata. Lines without '=' are skipped; duplicate names: last wins.
func Parse(r io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if name, value, ok := parseEntryLine(line); ok {
			values[name] = value
		}
	}
	return values, sc.Err()
}

// Load reads the file at path and returns its entries as a map (metadata is
// discarded; see LoadFile for the full view).
func Load(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// ---------------------------------------------------------------------------
// mutation
// ---------------------------------------------------------------------------

// Set creates or updates name. Created is only stamped when it is unknown;
// Modified always moves to now.
func (f *File) Set(name, value string, now time.Time) (*Entry, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := validateLineValue("value", value); err != nil {
		return nil, err
	}
	e, ok := f.index[name]
	if !ok {
		e = &Entry{Name: name}
		f.index[name] = e
		f.entries = append(f.entries, e)
		delete(f.deleted, name)
	}
	e.Value = value
	touch(e, now)
	return e, nil
}

// touch stamps Modified and, when the creation date is still unknown,
// Created as well.
func touch(e *Entry, now time.Time) {
	if e.Created.IsZero() {
		e.Created = now
	}
	e.Modified = now
}

// Delete removes name and its metadata. It reports whether the entry existed.
func (f *File) Delete(name string) bool {
	e, ok := f.index[name]
	if !ok {
		return false
	}
	delete(f.index, name)
	delete(f.lastEntryLine, name)
	f.deleted[name] = true
	for i, x := range f.entries {
		if x == e {
			f.entries = append(f.entries[:i], f.entries[i+1:]...)
			break
		}
	}
	return true
}

// SetNote adds or replaces a note on an existing entry and bumps Modified.
func (f *File) SetNote(name, label, value string, now time.Time) error {
	e, ok := f.index[name]
	if !ok {
		return fmt.Errorf("no such entry %q", name)
	}
	if err := validateNoteLabel(label); err != nil {
		return err
	}
	if err := validateLineValue("note value", value); err != nil {
		return err
	}
	e.Notes = setNote(e.Notes, label, value)
	touch(e, now)
	return nil
}

// DeleteNote removes a note and bumps Modified.
func (f *File) DeleteNote(name, label string, now time.Time) (bool, error) {
	e, ok := f.index[name]
	if !ok {
		return false, fmt.Errorf("no such entry %q", name)
	}
	notes, removed := deleteNote(e.Notes, label)
	if !removed {
		return false, nil
	}
	e.Notes = notes
	touch(e, now)
	return true, nil
}

// Backfill stamps metadata for entries that have none (typically entries
// written before metadata existed). It returns the number of entries changed.
func (f *File) Backfill(created, modified time.Time) int {
	n := 0
	for _, e := range f.entries {
		if !e.Created.IsZero() && !e.Modified.IsZero() {
			continue
		}
		if e.Created.IsZero() {
			e.Created = created
		}
		if e.Modified.IsZero() {
			e.Modified = modified
		}
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// rendering
// ---------------------------------------------------------------------------

// Bytes renders the file, preserving untouched lines verbatim. Duplicate
// entries collapse into the position of their last occurrence, and each
// entry is followed by its metadata lines.
func (f *File) Bytes() []byte {
	var b strings.Builder
	emitted := make(map[string]bool, len(f.entries))

	for i, line := range f.lines {
		info := f.infos[i]
		switch info.kind {
		case lineOther:
			b.WriteString(line)
			b.WriteByte('\n')
		case lineEntry:
			e, ok := f.index[info.name]
			if !ok {
				continue // deleted
			}
			if f.lastEntryLine[info.name] != i {
				continue // superseded duplicate
			}
			writeEntry(&b, e)
			emitted[info.name] = true
		case lineMeta, lineNote:
			if _, ok := f.index[info.name]; ok {
				continue // re-emitted with its entry
			}
			if f.deleted[info.name] {
				continue // its entry was deleted in this session
			}
			b.WriteString(line) // orphan (e.g. hand-edited): keep as a comment
			b.WriteByte('\n')
		}
	}

	for _, e := range f.entries {
		if !emitted[e.Name] {
			writeEntry(&b, e)
			emitted[e.Name] = true
		}
	}
	return []byte(b.String())
}

func writeEntry(b *strings.Builder, e *Entry) {
	b.WriteString(e.Name)
	b.WriteByte('=')
	b.WriteString(e.Value)
	b.WriteByte('\n')

	if !e.Created.IsZero() || !e.Modified.IsZero() {
		b.WriteString(MetaPrefix)
		b.WriteString(e.Name)
		if !e.Created.IsZero() {
			b.WriteString(" created=")
			b.WriteString(e.Created.Format(time.RFC3339))
		}
		if !e.Modified.IsZero() {
			b.WriteString(" modified=")
			b.WriteString(e.Modified.Format(time.RFC3339))
		}
		b.WriteByte('\n')
	}

	for _, n := range e.Notes {
		b.WriteString(NotePrefix)
		b.WriteString(e.Name)
		b.WriteByte(' ')
		b.WriteString(n.Label)
		b.WriteByte('=')
		b.WriteString(n.Value)
		b.WriteByte('\n')
	}
}

// Save writes the file atomically (temp file + rename) with mode 0600.
// Plaintext file: permissions are the only line of defense.
func (f *File) Save() error {
	if f.path == "" {
		return errors.New("secrets: no path set")
	}
	return f.SaveAs(f.path)
}

// SaveAs writes the rendered file to path atomically.
func (f *File) SaveAs(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".secrets-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(f.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	tmpName = ""
	f.path = path
	return nil
}

// IsNotExist reports whether err means "file not found" (os.IsNotExist wrapper
// so callers need not import io/fs).
func IsNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }
