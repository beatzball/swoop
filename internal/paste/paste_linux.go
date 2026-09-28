//go:build linux

package paste

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

const pasteKey = "ctrl+V"

// Copy puts text on the clipboard with wl-copy on Wayland or xclip on X,
// whichever is installed.
func Copy(text string) error {
	for _, c := range [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New("no clipboard tool: install wl-clipboard or xclip")
}

// Clipboard returns the text on the clipboard, with wl-paste on Wayland
// or xclip on X.
func Clipboard() (string, error) {
	for _, c := range [][]string{{"wl-paste", "--no-newline"}, {"xclip", "-selection", "clipboard", "-o"}} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		out, err := exec.Command(c[0], c[1:]...).Output()
		return string(out), err
	}
	return "", errors.New("no clipboard tool: install wl-clipboard or xclip")
}

// notify uses notify-send when it is installed, and is quiet otherwise.
func notify(note string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return nil
	}
	return exec.Command("notify-send", "swoop", note).Run()
}
