// Package protocol defines the one contract every swoop tool shares: the
// result line. A source prints lines, fzf shows them, and the picked line's
// id goes back to swoop-run and swoop-preview. The reasoning and the open
// questions live in the "Spec: the line protocol" issue; this file is the
// code form of that spec and must not grow a field the spec does not have.
package protocol

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Item is one result. Its fields map one-to-one onto the tab-separated line,
// in this order.
type Item struct {
	ID       string // opaque, stable, unique within the source; never shown
	Kind     string // app, file, command, url, text, or an extension's own word
	Icon     string // one glyph: emoji or Nerd Font codepoint; may be empty
	Title    string // the main text; fzf matches on this
	Subtitle string // dimmer text after the title; may be empty
}

// Fields is how many tab-separated fields a line has. fzf is told to hide
// the first two with --with-nth, so the count here and the fzf flags in
// bin/swoop must move together.
const Fields = 5

const sep = "\t"

// Line formats the item as one protocol line, without the trailing newline.
//
// A tab or newline inside a field would split the line, so they are replaced
// with spaces rather than rejected. A source must never fail to list an item
// because a file name contains a tab; a slightly wrong subtitle is better
// than a missing result.
func (it Item) Line() string {
	return strings.Join([]string{
		clean(it.ID), clean(it.Kind), clean(it.Icon), clean(it.Title), clean(it.Subtitle),
	}, sep)
}

var cleaner = strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

func clean(s string) string { return cleaner.Replace(s) }

// Parse reads one line back into an Item. It accepts a trailing newline.
func Parse(line string) (Item, error) {
	f := strings.Split(strings.TrimRight(line, "\r\n"), sep)
	if len(f) != Fields {
		return Item{}, fmt.Errorf("protocol: want %d fields, got %d", Fields, len(f))
	}
	if f[0] == "" {
		return Item{}, errors.New("protocol: empty id")
	}
	return Item{ID: f[0], Kind: f[1], Icon: f[2], Title: f[3], Subtitle: f[4]}, nil
}

// Write writes items as lines, one buffered writer and one flush at the end.
// swoop-list runs on keystrokes, and a write syscall per line is exactly the
// kind of cost that adds up to felt lag.
func Write(w io.Writer, items []Item) error {
	bw := bufio.NewWriter(w)
	for _, it := range items {
		if _, err := bw.WriteString(it.Line()); err != nil {
			return err
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}
