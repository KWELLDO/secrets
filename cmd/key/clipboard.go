package main

import (
	"os"
	"os/exec"
	"strings"
)

// writeClipboard puts value on the system clipboard.
//
// The platform-specific mechanism lives in writeClipboardOS, one file per OS
// (clipboard_unix.go / clipboard_darwin.go / clipboard_windows.go) so that
// commands which only exist on one platform — wl-copy on Linux, pbcopy on
// macOS, clip.exe on Windows — are never called from a foreign platform.
//
// KEY_CLIPBOARD overrides the whole detection and is honoured on every OS, so
// odd environments (WSL, no Wayland, a custom helper) can point at anything:
//
//	KEY_CLIPBOARD="xclip -selection clipboard" key copy foo
func writeClipboard(value string) error {
	if c := strings.TrimSpace(os.Getenv("KEY_CLIPBOARD")); c != "" {
		parts := strings.Fields(c)
		return runClipboardCommand(value, parts[0], parts[1:]...)
	}
	return writeClipboardOS(value)
}

// runClipboardCommand runs bin with args, feeding value on its stdin.
//
// Stdout is deliberately nil (which Go maps to os.DevNull). wl-copy forks a
// long-lived background child to serve the X11/Wayland selection; if that child
// inherited a pipe, the pipe would stay open and any Output/CombinedOutput
// call would block forever waiting for an EOF that never comes.
func runClipboardCommand(value, bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(value)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
