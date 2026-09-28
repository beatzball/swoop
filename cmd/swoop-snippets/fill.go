// The placeholders a snippet's text can hold, filled when it is pasted
// and when it is previewed:
//
//	{date}       today, 2026-09-28
//	{time}       now, 14:05
//	{clipboard}  what is on the clipboard
//	{uuid}       a new random UUID, version 4
//	{cursor}     where the caret is left after the paste
//
// Anything else in braces is text and stays as written, so a snippet of
// code with braces in it pastes as it is.
package main

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Env is where the placeholders get their values. The tool uses the real
// clock and clipboard; the tests use fixed ones.
type Env struct {
	Now       func() time.Time
	Clipboard func() (string, error)
	UUID      func() string
}

// placeholders are the names the text can hold, in braces.
var placeholders = []string{"date", "time", "clipboard", "uuid", "cursor"}

// Fill returns text with its placeholders filled, and how many characters
// of it come after {cursor}: the left-arrow presses that put the caret
// there once it is pasted. Only the first {cursor} counts; any others
// are removed.
//
// It is one pass from left to right, so what a placeholder puts in is
// never read again: a clipboard that holds "{date}" pastes as "{date}".
// The clipboard is read once, and only when the text asks for it, so a
// snippet without {clipboard} never waits on it; a clipboard that cannot
// be read fills in empty rather than stop the paste.
func Fill(text string, env Env) (out string, back int) {
	var b strings.Builder
	cursor := -1
	var now time.Time
	var clip *string
	for {
		i := strings.IndexByte(text, '{')
		if i < 0 {
			b.WriteString(text)
			break
		}
		b.WriteString(text[:i])
		text = text[i:]
		name, ok := placeholderAt(text)
		if !ok {
			b.WriteByte('{')
			text = text[1:]
			continue
		}
		text = text[len(name)+2:]
		switch name {
		case "date", "time":
			if now.IsZero() {
				now = env.Now()
			}
			if name == "date" {
				b.WriteString(now.Format("2006-01-02"))
			} else {
				b.WriteString(now.Format("15:04"))
			}
		case "clipboard":
			if clip == nil {
				s, err := env.Clipboard()
				if err != nil {
					s = ""
				}
				clip = &s
			}
			b.WriteString(*clip)
		case "uuid":
			b.WriteString(env.UUID())
		case "cursor":
			if cursor < 0 {
				cursor = b.Len()
			}
		}
	}
	out = b.String()
	if cursor >= 0 {
		// Characters, not bytes: one arrow press moves past one
		// character. An emoji built of several code points, a flag or a
		// family, is one press in most apps and several here; a caret
		// after one lands a little early.
		back = utf8.RuneCountInString(out[cursor:])
	}
	return out, back
}

// placeholderAt says which placeholder text starts with, if any.
func placeholderAt(text string) (string, bool) {
	for _, p := range placeholders {
		if strings.HasPrefix(text, "{"+p+"}") {
			return p, true
		}
	}
	return "", false
}

// realEnv is the clock, the clipboard and a random source.
func realEnv(clipboard func() (string, error)) Env {
	return Env{Now: time.Now, Clipboard: clipboard, UUID: newUUID}
}

// newUUID is a random version 4 UUID, as RFC 9562 lays it out.
func newUUID() string {
	var u [16]byte
	_, _ = rand.Read(u[:])
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
