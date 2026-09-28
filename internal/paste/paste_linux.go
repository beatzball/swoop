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

// keystroke is not there yet on Linux: which tool can type into the
// window in front differs between X and each Wayland compositor. The
// text is on the clipboard.
func keystroke(text string) (string, error) {
	return "Copied " + text + ". Paste it with " + pasteKey + "; swoop cannot paste for you on Linux yet.", nil
}

// notify uses notify-send when it is installed, and is quiet otherwise.
func notify(note string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return nil
	}
	return exec.Command("notify-send", "swoop", note).Run()
}
