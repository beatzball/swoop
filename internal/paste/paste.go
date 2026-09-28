// Package paste puts text where the user was typing: on the clipboard,
// then into the app in front with the paste keystroke. The emoji
// extension uses it for Enter, and snippets will. The keystroke is the
// per-OS part, one file each behind a build tag:
//
//   - macOS: cmd+V through System Events, which needs Accessibility for
//     the app swoop runs in. Without it, the text is copied and the user
//     is told where to turn it on
//   - Linux: copy only, with wl-copy or xclip, and a note. Pasting for
//     the user is a later step there
//
// A paste is sent only inside swoop's own frame (SWOOP_SHELL is set).
// In a plain terminal the app in front is that terminal, and cmd+V would
// land in the shell that follows the launcher, so there Paste copies and
// says so.
package paste

import (
	"fmt"
	"os"
)

// Paste copies text and, where it can, pastes it into the app in front.
// note is empty when the paste was sent. When only the copy happened, note
// says so and why, in words for the user; see Tell. err is for a copy
// that failed: nothing reached the clipboard.
func Paste(text string) (note string, err error) {
	if err := Copy(text); err != nil {
		return "", err
	}
	if !inFrame() {
		return "Copied " + text + ". Paste it with " + pasteKey + ".", nil
	}
	return keystroke(text)
}

// inFrame says whether swoop runs in its own frame, whose panel hides
// once the launcher exits and leaves the user's app in front.
func inFrame() bool { return os.Getenv("SWOOP_SHELL") != "" }

// Tell shows a note: on stderr, which a plain terminal shows once the
// launcher is gone, and inside the frame as a notification too, because
// the frame drops the terminal, and its stderr with it, as it hides.
func Tell(note string) {
	fmt.Fprintln(os.Stderr, note)
	if inFrame() {
		_ = notify(note)
	}
}
