//go:build windows

package main

import (
	"errors"
	"os/exec"
)

// writeClipboardOS pipes the value into clip.exe, the clipboard helper that
// ships with Windows.
//
// We deliberately stay on the "run the platform command" abstraction instead of
// calling the Win32 API through syscall: no cgo, no unsafe and no external
// module, and `go vet` stays clean on every platform.
//
// clip.exe decodes stdin with the active console code page, so ASCII values
// (API keys, tokens, URLs) are byte-exact. For non-ASCII values point
// KEY_CLIPBOARD at a Unicode-capable helper, e.g.
//
//	KEY_CLIPBOARD=powershell -NoProfile -Command Set-Clipboard
func writeClipboardOS(value string) error {
	p, err := exec.LookPath("clip")
	if err != nil {
		return errors.New("clip.exe not found (set KEY_CLIPBOARD to a clipboard helper)")
	}
	return runClipboardCommand(value, p)
}
