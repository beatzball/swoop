//go:build windows

package paste

import (
	"errors"
	"os/exec"
	"strings"
)

const pasteKey = "ctrl+V"

// Copy puts text on the clipboard with clip, which comes with Windows.
func Copy(text string) error {
	cmd := exec.Command("clip")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Missing is always empty: clip comes with Windows.
func Missing() string { return "" }

func Clipboard() (string, error) { return "", errors.New("not implemented on this OS yet") }

func notify(string) error { return nil }
