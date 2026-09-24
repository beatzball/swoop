// Package picture puts a PNG on the screen of a terminal that speaks the
// Kitty graphics protocol, which libghostty does. It is the one place swoop
// knows how a picture becomes bytes; swoop-img and swoop-preview both use
// it and neither imports the other.
//
// The picture is sent as a "virtual placement" and then drawn with Unicode
// placeholder cells. That is the mode fzf recommends for its preview pane:
// the placeholders are ordinary text, so fzf can redraw, scroll, and clear
// them like any other line, and a stale picture cannot be left behind. A
// direct placement would paint over the pane and survive the next redraw.
package picture

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"
)

// MaxCells is the largest column or row count a placeholder can address:
// the protocol encodes each with one combining diacritic from a fixed
// table of that length.
const MaxCells = 297

// cellAspect is a terminal cell's height divided by its width. Cells are
// about twice as tall as wide in every monospace font that matters, and
// the terminal will not tell a one-shot process the real number without a
// round trip, so this is assumed rather than measured.
const cellAspect = 2.0

// Fit returns the cell box a picture of pixel size w by h should occupy
// so that it keeps its shape and fits inside maxCols by maxRows. Both
// results are at least 1.
func Fit(w, h, maxCols, maxRows int) (cols, rows int) {
	if w <= 0 || h <= 0 || maxCols <= 0 || maxRows <= 0 {
		return 1, 1
	}
	// Try to use every row, then see if the width fits.
	rows = maxRows
	cols = int(float64(rows) * cellAspect * float64(w) / float64(h))
	if cols > maxCols {
		cols = maxCols
		rows = int(float64(cols) / cellAspect * float64(h) / float64(w))
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

// MaxID is the largest image id. A placeholder cell names its image with
// its foreground colour, and a 24-bit colour holds a 24-bit id.
const MaxID = 1<<24 - 1

// WritePNG sends png to the terminal as image id, scaled to cols by rows
// cells, and then writes rows lines of placeholder cells, each ending in a
// newline. This is the preview-pane shape: one picture, many rows. The id
// must be 1 to 255 here, because these cells name their image with a
// 256-colour foreground; see Cells for the 24-bit form used on result rows.
func WritePNG(w io.Writer, png []byte, cols, rows, id int) error {
	if id < 1 || id > 255 {
		return fmt.Errorf("picture: id %d is outside 1..255", id)
	}
	if err := Transmit(w, png, cols, rows, id); err != nil {
		return err
	}
	return placeholders(w, cols, rows, id)
}

// Cells returns one row of a picture as text: cols placeholder cells for
// row, with the image id in a 24-bit foreground colour, then a colour
// reset. It is what a result row carries for its icon: the picture itself
// was sent earlier with Transmit, and the terminal fills these cells in
// wherever they end up on screen. Anything that keeps the colour, such as
// fzf with --ansi, keeps the picture.
func Cells(id, row, cols int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", (id>>16)&0xff, (id>>8)&0xff, id&0xff)
	for c := 0; c < cols; c++ {
		b.WriteRune(kitty.Placeholder)
		b.WriteRune(kitty.Diacritic(row))
		b.WriteRune(kitty.Diacritic(c))
	}
	b.WriteString("\x1b[39m")
	return b.String()
}

// Transmit sends the PNG bytes to the terminal as image id, to be shown
// scaled to cols by rows cells wherever placeholder cells for that id
// appear. It writes in chunks, as the protocol requires: base64, at most
// kitty.MaxChunkSize bytes per escape, m=1 on every chunk but the last.
// The first chunk carries the options; a=T transmits and places in one go,
// U=1 makes the placement virtual, and q=2 tells the terminal not to
// answer, since nothing here is reading the reply.
//
// Note that a terminal keeps pictures per screen. A picture sent on the
// normal screen is not there once fzf switches to the alternate screen, so
// a caller that feeds fzf must send after fzf has started.
func Transmit(w io.Writer, png []byte, cols, rows, id int) error {
	if id < 1 || id > MaxID {
		return fmt.Errorf("picture: id %d is outside 1..%d", id, MaxID)
	}
	if cols < 1 || rows < 1 || cols > MaxCells || rows > MaxCells {
		return fmt.Errorf("picture: %d by %d cells is outside 1..%d", cols, rows, MaxCells)
	}
	data := base64.StdEncoding.EncodeToString(png)
	opts := fmt.Sprintf("a=T,f=100,i=%d,c=%d,r=%d,U=1,q=2", id, cols, rows)
	for len(data) > 0 {
		n := len(data)
		if n > kitty.MaxChunkSize {
			n = kitty.MaxChunkSize
		}
		chunk, rest := data[:n], data[n:]
		more := 1
		if len(rest) == 0 {
			more = 0
		}
		var err error
		if opts != "" {
			_, err = fmt.Fprintf(w, "\x1b_G%s,m=%d;%s\x1b\\", opts, more, chunk)
			opts = ""
		} else {
			_, err = fmt.Fprintf(w, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
		if err != nil {
			return err
		}
		data = rest
	}
	return nil
}

// placeholders writes the cells the terminal will paint the picture into.
// Each cell is the placeholder rune followed by two diacritics that say
// which row and column of the picture it shows, with the image id in the
// foreground colour. Every cell is spelled out in full rather than relying
// on the protocol's "same as the cell to the left" shortcut, so a line that
// fzf cuts or wraps still shows the right piece.
func placeholders(w io.Writer, cols, rows, id int) error {
	var b strings.Builder
	for r := 0; r < rows; r++ {
		fmt.Fprintf(&b, "\x1b[38;5;%dm", id)
		for c := 0; c < cols; c++ {
			b.WriteRune(kitty.Placeholder)
			b.WriteRune(kitty.Diacritic(r))
			b.WriteRune(kitty.Diacritic(c))
		}
		b.WriteString("\x1b[39m\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}
