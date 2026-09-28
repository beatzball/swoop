// Package paste puts text where the user was typing: on the clipboard,
// then into the app in front with the paste keystroke. The emoji and
// snippets extensions use it for Enter. The keystroke is the
// per-OS part, one file each behind a build tag:
//
//   - macOS: cmd+V through System Events, which needs Accessibility for
//     the app swoop runs in. Without it, the text is copied and the user
//     is told where to turn it on
//   - Linux: copy only, with wl-copy or xclip, and a note. Pasting for
//     the user is a later step there
//
// After the paste, the caret can be moved back with left-arrow presses,
// for a snippet with {cursor} in it. That rides on the same keystroke
// and the same fallback: where there is no paste, there are no arrows.
//
// A paste is sent only inside swoop's own frame (SWOOP_SHELL is set).
// In a plain terminal the app in front is that terminal, and cmd+V would
// land in the shell that follows the launcher, so there Paste copies and
// says so.
package paste

import (
	"fmt"
	"os"
	"strings"
)

// Paste copies text and, where it can, pastes it into the app in front.
// note is empty when the paste was sent. When only the copy happened, note
// says so and why, in words for the user; see Tell. err is for a copy
// that failed: nothing reached the clipboard.
func Paste(text string) (note string, err error) { return PasteBack(text, 0) }

// PasteBack is Paste, then left presses of the left arrow key, which
// leave the caret that many characters before the end of the text.
func PasteBack(text string, left int) (note string, err error) {
	if err := Copy(text); err != nil {
		return "", err
	}
	if !inFrame() {
		return "Copied " + Short(text) + ". Paste it with " + pasteKey + ".", nil
	}
	return keystroke(text, max(left, 0))
}

// Short is text cut to fit in a note: the first line, 40 characters at
// most. A snippet can be pages long; the note only has to say which.
func Short(text string) string {
	line, _, cut := strings.Cut(text, "\n")
	if r := []rune(line); len(r) > 40 {
		line, cut = string(r[:40]), true
	}
	if cut {
		line += "…"
	}
	return line
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
