package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gitee.com/kwelldo/secrets"
)

// confirmDelete asks on stderr whether to delete e and reads the answer from
// the terminal. Non-interactive callers skip it (see cmdRm).
func confirmDelete(e *secrets.Entry) bool {
	what := e.Name
	if n := len(e.Notes); n > 0 {
		word := "notes"
		if n == 1 {
			word = "note"
		}
		what = fmt.Sprintf("%s (with %d %s)", e.Name, n, word)
	}
	fmt.Fprintf(os.Stderr, "delete %s? [y/N] ", what)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

// trashPath returns the snapshot file for the secrets file at path. It holds
// one entry in the standard format: the most recently deleted one.
func trashPath(path string) string { return path + ".trash" }

func writeTrash(path string, data []byte) error {
	return os.WriteFile(trashPath(path), data, 0o600)
}

// cmdUndo restores the entry that `key rm` deleted last.
func cmdUndo(path string, args []string) {
	if len(args) != 0 {
		usageError("usage: key undo")
	}
	trash := trashPath(path)
	b, err := os.ReadFile(trash)
	if os.IsNotExist(err) {
		fatal("nothing to undo (no %s)", trash)
	}
	if err != nil {
		fatal("read %s: %v", trash, err)
	}
	tf, err := secrets.ParseFile(strings.NewReader(string(b)))
	if err != nil {
		fatal("parse %s: %v", trash, err)
	}
	if len(tf.Entries()) != 1 {
		fatal("%s is corrupt: want exactly one entry", trash)
	}
	e := tf.Entries()[0]

	f := load(path)
	if err := f.Restore(*e); err != nil {
		fatal("%v", err)
	}
	save(f)
	if err := os.Remove(trash); err != nil {
		fatal("remove %s: %v", trash, err)
	}
	fmt.Printf("restored %s\n", e.Name)
}
