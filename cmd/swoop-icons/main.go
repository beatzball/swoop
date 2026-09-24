// swoop-icons turns the glyph on each app row into the app's real icon. It
// reads result lines on stdin and writes them back on stdout with the icon
// field replaced by two picture cells. The pictures themselves go
// elsewhere: to the file named by -out, which bin/swoop sends to the
// terminal once fzf has started, because a terminal keeps pictures per
// screen and fzf draws on the alternate one.
//
// Rows whose icon cannot be found keep their glyph, padded to the same two
// columns, so titles stay aligned. If -out cannot be opened, every row
// keeps its glyph and the list still works.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	"github.com/beatzball/swoop/internal/bundle"
	"github.com/beatzball/swoop/internal/picture"
	"github.com/beatzball/swoop/internal/protocol"
)

// iconCols and iconRows are the cell box of a row icon. Two columns by one
// row is close to square in any terminal font.
const iconCols, iconRows = 2, 1

// iconPx is the pixel size of a row icon. Two columns are under 50 device
// pixels on a high-density display; 64 is enough and keeps the startup
// send small.
const iconPx = 64

func main() {
	out := flag.String("out", "/dev/tty", "where the pictures go")
	flag.Parse()

	var pics io.Writer
	if f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600); err == nil {
		defer f.Close()
		bw := bufio.NewWriterSize(f, 64<<10)
		defer bw.Flush()
		pics = bw
	}

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64<<10), 1<<20)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for in.Scan() {
		line := in.Text()
		it, err := protocol.Parse(line)
		if err != nil {
			// Not ours to judge; pass it through untouched.
			fmt.Fprintln(w, line)
			continue
		}
		it.Icon = icon(it, pics)
		fmt.Fprintln(w, it.Line())
	}
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-icons:", err)
		os.Exit(1)
	}
}

// icon returns the two-column icon field for it: picture cells when the
// app's icon can be sent, otherwise the glyph it came with, padded.
func icon(it protocol.Item, pics io.Writer) string {
	fallback := it.Icon + " "
	if pics == nil || it.Kind != "app" {
		return fallback
	}
	data, err := bundle.Describe(it.ID).IconPNG(iconPx)
	if err != nil {
		return fallback
	}
	// One id per app, from its path, so the same app always lands in the
	// same slot and a second run replaces rather than piles up.
	id := 1 + int(crc32.ChecksumIEEE([]byte(it.ID))%picture.MaxID)
	if err := picture.Transmit(pics, data, iconCols, iconRows, id); err != nil {
		return fallback
	}
	return picture.Cells(id, 0, iconCols)
}
