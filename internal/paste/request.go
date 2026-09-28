package paste

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// envRequest names the file a frame of our own reads when the launcher
// exits. The frame sets it for the launcher it runs, and the launcher
// writes a paste request there instead of pressing cmd+V itself: the
// frame is the one process that knows when its panel is off screen, and
// the one that holds the Accessibility grant.
const envRequest = "SWOOP_PASTE"

// The request file is two parts, and nothing else:
//
//	2
//	Best,
//	Sam
//
// The first line is how many times to press the left arrow after the
// paste, in decimal, 0 for none: that is how a snippet's {cursor} puts
// the caret back. Everything after that first newline is the text, UTF-8,
// byte for byte, newlines and all; no newline is added at the end. The
// count goes first so the text can hold any line at all.
//
// The frame, when the launcher tells it it is leaving (SIGUSR2), hides
// its panel, reads and removes the file, and pastes the text if a text
// field is in front; otherwise the text stays on the clipboard and a
// notification says so. The clipboard already holds the text either way.

// Request is what the frame is asked to paste.
type Request struct {
	Text string
	Left int
}

// Encode is the request as the file holds it.
func (r Request) Encode() []byte {
	return []byte(strconv.Itoa(max(r.Left, 0)) + "\n" + r.Text)
}

// DecodeRequest reads a request file's bytes.
func DecodeRequest(data []byte) (Request, error) {
	head, text, ok := strings.Cut(string(data), "\n")
	if !ok {
		return Request{}, errors.New("paste request: no count line")
	}
	left, err := strconv.Atoi(head)
	if err != nil || left < 0 {
		return Request{}, errors.New("paste request: bad count " + strconv.Quote(head))
	}
	return Request{Text: text, Left: left}, nil
}

// WriteRequest writes r to path whole, through a file beside it and a
// rename, so the frame never reads half a request.
func WriteRequest(path string, r Request) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, r.Encode(), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
