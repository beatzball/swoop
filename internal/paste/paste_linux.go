//go:build linux

package paste

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/beatzball/swoop/internal/tool"
)

const pasteKey = "ctrl+V"

// copiers are the clipboard tools Copy tries, in order: wl-copy on
// Wayland, xclip on X. Missing reads the same list, so the two agree.
var copiers = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}}

// noTool is what Copy fails with, and what Missing says, when neither is
// installed: a bare Linux install often has neither.
const noTool = "no clipboard tool: install wl-clipboard or xclip"

// Copy puts text on the clipboard with wl-copy on Wayland or xclip on X,
// whichever is installed.
func Copy(text string) error {
	for _, c := range copiers {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New(noTool)
}

// Missing says, in words for a row's subtitle, why Copy cannot work
// here, before anyone presses Enter to find out. Empty when it can.
func Missing() string {
	for _, c := range copiers {
		if _, err := exec.LookPath(c[0]); err == nil {
			return ""
		}
	}
	return noTool
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
	return exec.Command("notify-send", tool.Read().Title, note).Run()
}
