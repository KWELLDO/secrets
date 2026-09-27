//go:build linux || freebsd || netbsd || openbsd || dragonfly || solaris

package main

import (
	"errors"
	"os"
	"os/exec"
)

// writeClipboardOS picks the X11/Wayland clipboard helper that is actually
// installed. Wayland is tried first when a Wayland session is detected, then
// X11 (xclip, then xsel), then wl-copy as a last resort for sessions that do
// not export the usual environment variables.
func writeClipboardOS(value string) error {
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		if p, err := exec.LookPath("wl-copy"); err == nil {
			return runClipboardCommand(value, p)
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if p, err := exec.LookPath("xclip"); err == nil {
			return runClipboardCommand(value, p, "-selection", "clipboard")
		}
		if p, err := exec.LookPath("xsel"); err == nil {
			return runClipboardCommand(value, p, "--clipboard", "--input")
		}
	}
	if p, err := exec.LookPath("wl-copy"); err == nil {
		return runClipboardCommand(value, p)
	}
	return errors.New("no clipboard tool found (install wl-clipboard or xclip/xsel, or set KEY_CLIPBOARD)")
}
