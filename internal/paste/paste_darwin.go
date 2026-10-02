//go:build darwin

package paste

import (
	"os"
	"os/exec"
	"strings"

	"github.com/beatzball/swoop/internal/tool"
)

const pasteKey = "cmd+V"

// Copy puts text on the clipboard.
func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Missing is always empty: pbcopy comes with macOS.
func Missing() string { return "" }

// Clipboard returns the text on the clipboard.
func Clipboard() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	return string(out), err
}

// notify posts a notification titled with the tool's title.
func notify(note string) error {
	cmd := exec.Command("osascript", "-e", notification(note))
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func notification(note string) string {
	return "display notification " + appleString(note) + " with title " + appleString(tool.Read().Title)
}

// appleString quotes s as an AppleScript string literal.
func appleString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
