// Package paste puts text where the user was typing: on the clipboard,
// then into the app in front with the paste keystroke. The emoji and
// snippets extensions use it for Enter.
//
// The keystroke is not sent from here. Inside the launcher's own frame
// (SWOOP_SHELL is set, and the frame names a request file in
// SWOOP_PASTE), Paste copies, writes the text to that file, and returns;
// the frame, as the launcher exits, hides its panel and pastes, but only
// into a text field. The frame owns the moment the panel is gone and the
// Accessibility grant, so the keystroke is its job; the file format is in
// request.go.
//
// Everywhere else the text is only copied, and the note says to paste by
// hand: in a plain terminal the app in front is that terminal, and cmd+V
// would land in the shell that follows the launcher. The clipboard is the
// per-OS part, one file each behind a build tag: pbcopy on macOS, wl-copy
// or xclip on Linux, clip on Windows.
//
// After the paste, the caret can be moved back with left-arrow presses,
// for a snippet with {cursor} in it. The count rides in the same request;
// where there is no paste, there are no arrows.
package paste

import (
	"fmt"
	"os"
	"strings"
)

// Paste copies text and, inside the frame, asks the frame to paste it
// into the app in front. note is empty when the frame was asked: the
// frame tells the user if it only copied. Otherwise note says the text
// was copied, in words for the user; see Tell. err is for a copy that
// failed, or a request that could not be written.
func Paste(text string) (note string, err error) { return PasteBack(text, 0) }

// PasteBack is Paste, then left presses of the left arrow key, which
// leave the caret that many characters before the end of the text.
func PasteBack(text string, left int) (note string, err error) {
	if err := Copy(text); err != nil {
		return "", err
	}
	if path := os.Getenv(envRequest); inFrame() && path != "" {
		return "", WriteRequest(path, Request{Text: text, Left: max(left, 0)})
	}
	return "Copied " + Short(text) + ". Paste it with " + pasteKey + ".", nil
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

// inFrame says whether the launcher runs in its own frame, whose panel hides
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
