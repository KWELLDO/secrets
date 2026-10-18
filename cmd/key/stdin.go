package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// readSecretValue returns the value for `key set <name>` taken from stdin.
// On a terminal it prompts with echo off, so the value never lands in the
// shell history; on a pipe it reads one line (a trailing newline is dropped).
func readSecretValue(name string) (string, error) {
	if isTerminal(os.Stdin) {
		return readPassword(name)
	}
	return readPipedValue(os.Stdin)
}

// isTerminal reports whether f is a character device (a terminal), as opposed
// to a pipe or a regular file.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func readPassword(name string) (string, error) {
	fmt.Fprintf(os.Stderr, "value for %s: ", name)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // end the prompt line; ReadPassword eats no newline
	if err != nil {
		return "", fmt.Errorf("read value: %w", err)
	}
	if v := strings.TrimSpace(string(b)); v != "" {
		return v, nil
	}
	return "", errors.New("no value given")
}

// readPipedValue reads one line from r: a single trailing newline (with an
// optional '\r' before it) is dropped and surrounding whitespace is trimmed,
// matching what the file parser itself will see.
func readPipedValue(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read value: %w", err)
	}
	v := strings.TrimSuffix(string(b), "\n")
	v = strings.TrimSuffix(v, "\r")
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("no value given")
	}
	if strings.ContainsAny(v, "\r\n") {
		return "", errors.New("piped value must be a single line")
	}
	return v, nil
}
