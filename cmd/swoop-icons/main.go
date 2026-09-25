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
//
// Only icons already converted to PNG are sent. Converting one costs about
// 5 ms, and the very first run on a machine would spend half a second on a
// hundred apps before fzf could start. So an icon that is not cached yet
// keeps its glyph on this run, and swoop-icons starts a copy of itself in
// the background, `swoop-icons -warm <app>...`, to convert them. The next
// run has them.
package main

import (
	"bufio"
	"errors"
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
	warmMode := flag.Bool("warm", false, "convert the icons of the apps named as arguments, then exit")
	flag.Parse()

	if *warmMode {
		if err := warm(flag.Args()); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-icons:", err)
			os.Exit(1)
		}
		return
	}

	var pics io.Writer
	if f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600); err == nil {
		defer f.Close()
		bw := bufio.NewWriterSize(f, 64<<10)
		defer bw.Flush()
		pics = bw
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	missing, err := run(os.Stdin, w, pics, cachedIcon)
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-icons:", err)
		os.Exit(1)
	}
	if len(missing) > 0 {
		// The rows are already written; a warmer that cannot start costs
		// nothing but the icons, and this run is not the place to say so.
		_ = startWarmer(missing)
	}
}

// cachedIcon is the icon lookup swoop-icons uses: the PNG from the cache,
// or bundle.ErrNotCached.
func cachedIcon(app string) ([]byte, error) {
	return bundle.Describe(app).CachedIconPNG(iconPx)
}

// run copies result lines from r to w with each app's icon field replaced,
// asking lookup for the PNG of each app. It returns the apps whose icon
// lookup reported bundle.ErrNotCached: those kept their glyph this time,
// and are the ones worth warming. An app with no icon at all is not among
// them, because no amount of warming will give it one.
func run(r io.Reader, w io.Writer, pics io.Writer, lookup func(app string) ([]byte, error)) ([]string, error) {
	in := bufio.NewScanner(r)
	in.Buffer(make([]byte, 0, 64<<10), 1<<20)
	var missing []string
	for in.Scan() {
		line := in.Text()
		it, err := protocol.Parse(line)
		if err != nil {
			// Not ours to judge; pass it through untouched.
			fmt.Fprintln(w, line)
			continue
		}
		var notCached bool
		it.Icon, notCached = icon(it, pics, lookup)
		if notCached {
			missing = append(missing, it.ID)
		}
		fmt.Fprintln(w, it.Line())
	}
	return missing, in.Err()
}

// icon returns the two-column icon field for it: picture cells when the
// app's icon can be sent, otherwise the glyph it came with, padded. The
// second result is true when the glyph is there only because the icon is
// not cached yet.
func icon(it protocol.Item, pics io.Writer, lookup func(app string) ([]byte, error)) (string, bool) {
	fallback := it.Icon + " "
	if pics == nil || it.Kind != "app" {
		return fallback, false
	}
	data, err := lookup(it.ID)
	if err != nil {
		return fallback, errors.Is(err, bundle.ErrNotCached)
	}
	// One id per app, from its path, so the same app always lands in the
	// same slot and a second run replaces rather than piles up.
	id := 1 + int(crc32.ChecksumIEEE([]byte(it.ID))%picture.MaxID)
	if err := picture.Transmit(pics, data, iconCols, iconRows, id); err != nil {
		return fallback, false
	}
	return picture.Cells(id, 0, iconCols), false
}
