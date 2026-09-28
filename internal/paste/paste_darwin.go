//go:build darwin

package paste

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const pasteKey = "cmd+V"

// Copy puts text on the clipboard.
func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// delay is how long the keystroke waits: long enough for the launcher to
// exit and the frame to put its panel away, so cmd+V reaches the app
// that was in front, not the panel. The frame hides within a few ms of
// the signal; a quarter second leaves room for a busy machine.
const delay = "0.25"

// Clipboard returns the text on the clipboard.
func Clipboard() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	return string(out), err
}

// keystrokeScript is the AppleScript for cmd+V, then left presses of the
// left arrow (key code 123). The short delay lets the app take the paste
// in before the arrows arrive, so they move the caret in the pasted text
// and not before it.
func keystrokeScript(left int) []string {
	lines := []string{`tell application "System Events"`, `keystroke "v" using command down`}
	if left > 0 {
		lines = append(lines, "delay 0.1", "repeat "+strconv.Itoa(left)+" times", "key code 123", "end repeat")
	}
	return append(lines, "end tell")
}

// keystroke sends cmd+V from a process of its own, in a session of its
// own. It has to outlive this one: the panel is still up while the
// launcher runs this, and the frame ends everything in the terminal once
// it hides. If System Events refuses (Automation, the first time, or
// Accessibility taken away since the check), a notification says so.
func keystroke(text string, left int) (string, error) {
	if !trusted() {
		return "Copied " + Short(text) + ", not pasted: swoop needs Accessibility to paste for you. " +
			"Turn on swoop-shell-mac in System Settings > Privacy & Security > Accessibility.", nil
	}
	var args strings.Builder
	for _, line := range keystrokeScript(left) {
		args.WriteString(" -e " + quote(line))
	}
	script := "sleep " + delay + "; osascript" + args.String() + " >/dev/null 2>&1 || " +
		"osascript -e " + quote(notification("Copied, not pasted: allow swoop-shell-mac to control System Events in System Settings > Privacy & Security > Automation.")) + " >/dev/null 2>&1"
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return "", cmd.Process.Release()
}

// trusted asks macOS whether the app swoop runs in may send keystrokes.
// osascript inherits that app's permission, since it runs under it, so
// its answer is the one that counts. 100 ms, paid once on Enter.
func trusted() bool {
	out, err := exec.Command("osascript", "-l", "JavaScript", "-e", "ObjC.import('ApplicationServices'); $.AXIsProcessTrusted()").Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// notify posts a notification titled swoop.
func notify(note string) error {
	cmd := exec.Command("osascript", "-e", notification(note))
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func notification(note string) string {
	return "display notification " + appleString(note) + ` with title "swoop"`
}

// appleString quotes s as an AppleScript string literal.
func appleString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// quote single-quotes s for /bin/sh.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
