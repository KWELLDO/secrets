//go:build darwin

package main

import (
	"errors"
	"os/exec"
)

// writeClipboardOS uses pbcopy, which ships with macOS.
func writeClipboardOS(value string) error {
	p, err := exec.LookPath("pbcopy")
	if err != nil {
		return errors.New("pbcopy not found (it normally ships with macOS; set KEY_CLIPBOARD to override)")
	}
	return runClipboardCommand(value, p)
}
