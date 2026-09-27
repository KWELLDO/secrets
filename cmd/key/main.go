// Command key reads, writes and annotates entries in the flat ~/.secrets file.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"gitee.com/kwelldo/secrets"
)

const envFile = "SECRETS_FILE"

func defaultPath() string {
	if p := os.Getenv(envFile); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".secrets")
	}
	return filepath.Join(home, ".secrets")
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "key: "+format+"\n", a...)
	os.Exit(1)
}

func usageError(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "key: "+format+"\n", a...)
	os.Exit(2)
}

func usage() {
	fmt.Fprint(os.Stderr, `key - minimal flat secrets tool

Usage:
  key path                          print the secrets file path
  key list [-l|--long]              list key names (with -l: dates and notes)
  key get <name>                    print the value of <name> (exit 1 if absent)
  key copy <name>                   copy the value of <name> to the clipboard
  key info <name>                   print value, dates and notes of <name>
  key set <name>=<value>            create or update an entry
  key rm <name>                     delete an entry (and its notes)
  key note <name>                   list the notes of <name>
  key note <name> <label>=<value>   add or replace a note
  key unnote <name> <label>         remove a note
  key backfill [<time>]             stamp dates on entries that have none

Format: one name=value per line, '#' starts a comment, first '=' splits.
Metadata lives in comment lines so every flat parser still works:

  #@meta <name> created=<RFC3339> modified=<RFC3339>
  #@note <name> <label>=<value>

Notes are free-form labels; useful ones: base_url, account, url, email, note.
Override the default ~/.secrets path with the SECRETS_FILE environment variable.
See ~/Documents/secrets-format-spec.md for the full spec.

Examples:
  key set deepseek_api_key=sk-xxx
  key note deepseek_api_key base_url=https://api.deepseek.com
  key note deepseek_api_key account=kwelldo
  key info deepseek_api_key
  key copy deepseek_api_key
  key rm deepseek_api_key`)
}

// load returns the parsed file; a missing file yields an empty one so that
// mutating commands can create it on first use.
func load(path string) *secrets.File {
	f, err := secrets.LoadFile(path)
	if err == nil {
		return f
	}
	if secrets.IsNotExist(err) {
		return secrets.NewFile(path)
	}
	fatal("read %s: %v", path, err)
	return nil
}

func save(f *secrets.File) {
	if err := f.Save(); err != nil {
		fatal("write %s: %v", f.Path(), err)
	}
}

func ts(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func rel(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	future := d < 0
	if future {
		d = -d
	}
	switch {
	case d < time.Minute:
		if future {
			return "in the future"
		}
		return "just now"
	case d < time.Hour:
		return wrapAge(fmt.Sprintf("%dm", int(d.Minutes())), future)
	case d < 24*time.Hour:
		return wrapAge(fmt.Sprintf("%dh", int(d.Hours())), future)
	default:
		return wrapAge(fmt.Sprintf("%dd", int(d.Hours()/24)), future)
	}
}

func wrapAge(s string, future bool) string {
	if future {
		return "in " + s
	}
	return s + " ago"
}

func noteLabels(notes []secrets.Note) string {
	if len(notes) == 0 {
		return "-"
	}
	labels := make([]string, 0, len(notes))
	for _, n := range notes {
		labels = append(labels, n.Label)
	}
	return strings.Join(labels, ",")
}

func cmdList(path string, args []string) {
	long := false
	for _, a := range args {
		switch a {
		case "-l", "--long", "--all":
			long = true
		default:
			usageError("list: unexpected argument %q", a)
		}
	}

	f := load(path)
	entries := f.SortedEntries()
	if len(entries) == 0 {
		return
	}
	if !long {
		for _, e := range entries {
			fmt.Println(e.Name)
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCREATED\tMODIFIED\tNOTES")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name, ts(e.Created), ts(e.Modified), noteLabels(e.Notes))
	}
	w.Flush()
}

func cmdGet(path string, args []string) {
	if len(args) != 1 {
		usageError("usage: key get <name>")
	}
	f := load(path)
	e := f.Entry(args[0])
	if e == nil {
		fmt.Fprintf(os.Stderr, "key: %q not found in %s\n", args[0], path)
		os.Exit(1)
	}
	fmt.Println(e.Value)
}

func cmdCopy(path string, args []string) {
	if len(args) != 1 {
		usageError("usage: key copy <name>")
	}
	f := load(path)
	e := f.Entry(args[0])
	if e == nil {
		fmt.Fprintf(os.Stderr, "key: %q not found in %s\n", args[0], path)
		os.Exit(1)
	}
	if err := writeClipboard(e.Value); err != nil {
		fatal("clipboard: %v", err)
	}
	fmt.Printf("copied %s (%d chars)\n", e.Name, len(e.Value))
}

func cmdInfo(path string, args []string) {
	if len(args) != 1 {
		usageError("usage: key info <name>")
	}
	f := load(path)
	e := f.Entry(args[0])
	if e == nil {
		fmt.Fprintf(os.Stderr, "key: %q not found in %s\n", args[0], path)
		os.Exit(1)
	}
	fmt.Printf("name:      %s\n", e.Name)
	fmt.Printf("value:     %s\n", e.Value)
	fmt.Printf("created:   %s\n", dateCell(e.Created))
	fmt.Printf("modified:  %s\n", dateCell(e.Modified))
	if len(e.Notes) == 0 {
		fmt.Printf("notes:     -\n")
		return
	}
	fmt.Printf("notes:\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, n := range e.Notes {
		fmt.Fprintf(w, "  %s\t%s\n", n.Label, n.Value)
	}
	w.Flush()
}

func dateCell(t time.Time) string {
	if t.IsZero() {
		return "- (unknown)"
	}
	if r := rel(t); r != "" {
		return ts(t) + " (" + r + ")"
	}
	return ts(t)
}

func splitAssignment(arg string) (name, value string) {
	name, value, ok := strings.Cut(arg, "=")
	if !ok || strings.TrimSpace(name) == "" {
		usageError("invalid argument %q (want name=value)", arg)
	}
	return strings.TrimSpace(name), value
}

func cmdSet(path string, args []string) {
	if len(args) != 1 {
		usageError("usage: key set <name>=<value>")
	}
	name, value := splitAssignment(args[0])
	f := load(path)
	if _, err := f.Set(name, value, time.Now()); err != nil {
		fatal("%v", err)
	}
	save(f)
}

func cmdRm(path string, args []string) {
	if len(args) != 1 {
		usageError("usage: key rm <name>")
	}
	f := load(path)
	if !f.Delete(args[0]) {
		fmt.Fprintf(os.Stderr, "key: %q not found in %s\n", args[0], path)
		os.Exit(1)
	}
	save(f)
}

func cmdNote(path string, args []string) {
	if len(args) < 1 || len(args) > 2 {
		usageError("usage: key note <name> [<label>=<value>]")
	}
	f := load(path)
	name := args[0]
	e := f.Entry(name)
	if e == nil {
		fmt.Fprintf(os.Stderr, "key: %q not found in %s\n", name, path)
		os.Exit(1)
	}

	if len(args) == 1 { // list notes
		if len(e.Notes) == 0 {
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, n := range e.Notes {
			fmt.Fprintf(w, "%s\t%s\n", n.Label, n.Value)
		}
		w.Flush()
		return
	}

	label, value := splitAssignment(args[1])
	if err := f.SetNote(name, label, value, time.Now()); err != nil {
		fatal("%v", err)
	}
	save(f)
}

func cmdUnnote(path string, args []string) {
	if len(args) != 2 {
		usageError("usage: key unnote <name> <label>")
	}
	f := load(path)
	removed, err := f.DeleteNote(args[0], args[1], time.Now())
	if err != nil {
		fatal("%v", err)
	}
	if !removed {
		fmt.Fprintf(os.Stderr, "key: %q has no note %q\n", args[0], args[1])
		os.Exit(1)
	}
	save(f)
}

func cmdBackfill(path string, args []string) {
	if len(args) > 1 {
		usageError("usage: key backfill [<time>]")
	}
	at := time.Now()
	if len(args) == 1 {
		t, err := parseTime(args[0])
		if err != nil {
			usageError("%v", err)
		}
		at = t
	} else if st, err := os.Stat(path); err == nil {
		at = st.ModTime()
	}

	f := load(path)
	n := f.Backfill(at, at)
	if n == 0 {
		fmt.Fprintln(os.Stderr, "key: nothing to backfill")
		return
	}
	save(f)
	fmt.Printf("backfilled %d entr%s with %s\n", n, plural(n), ts(at))
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q (want RFC3339, \"2006-01-02 15:04:05\" or \"2006-01-02\")", s)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	path := defaultPath()
	switch args[0] {
	case "path":
		fmt.Println(path)
	case "list", "ls":
		cmdList(path, args[1:])
	case "get":
		cmdGet(path, args[1:])
	case "copy", "cp":
		cmdCopy(path, args[1:])
	case "info", "meta", "show", "describe":
		cmdInfo(path, args[1:])
	case "set":
		cmdSet(path, args[1:])
	case "rm", "del", "delete", "unset":
		cmdRm(path, args[1:])
	case "note", "notes":
		cmdNote(path, args[1:])
	case "unnote":
		cmdUnnote(path, args[1:])
	case "backfill":
		cmdBackfill(path, args[1:])
	case "help", "-h", "--help":
		usage()
	default:
		usageError("unknown command %q (try: key help)", args[0])
	}
}
