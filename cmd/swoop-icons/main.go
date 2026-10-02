// swoop-icons turns the glyph on a row into the icon of the file the row
// names. It reads result lines on stdin and writes them back on stdout
// with the icon field replaced by two picture cells. The pictures
// themselves go elsewhere: to the file named by -out, which bin/swoop
// sends to the terminal once fzf has started, because a terminal keeps
// pictures per screen and fzf draws on the alternate one.
//
// Every row on stdin is one whose id is the path of a file: swoop-nav
// sends only the rows of an extension that asked, with a file named
// "icons" beside it (see ext.IconsFile). The launcher's prefix on the id,
// when it is there, is not part of the path. What a file's icon is, is
// internal/bundle's to say; today that is an application bundle's on
// macOS.
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
	"github.com/beatzball/swoop/internal/ext"
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
	warmMode := flag.Bool("warm", false, "convert the icons of the files named as arguments, then exit")
	flag.Parse()

	if *warmMode {
		if err := warm(flag.Args()); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-icons:", err)
			os.Exit(1)
		}
		return
	}

	// With pics nil every row keeps its glyph and no warmer starts: a
	// terminal that cannot draw the pictures has no use for the cache.
	var pics io.Writer
	if !picture.Enabled() {
		// Nothing to send.
	} else if f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600); err == nil {
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
func cachedIcon(file string) ([]byte, error) {
	return bundle.Describe(file).CachedIconPNG(iconPx)
}

// run copies result lines from r to w with each row's icon field replaced,
// asking lookup for the PNG of the file each row names. It returns the
// files whose icon lookup reported bundle.ErrNotCached: those kept their
// glyph this time, and are the ones worth warming. A file with no icon at
// all is not among them, because no amount of warming will give it one.
func run(r io.Reader, w io.Writer, pics io.Writer, lookup func(file string) ([]byte, error)) ([]string, error) {
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
		// The file is the id as the extension printed it, without the
		// launcher's prefix.
		file := it.ID
		if _, raw, ok := ext.Route(it.ID); ok {
			file = raw
		}
		var notCached bool
		it.Icon, notCached = icon(it, file, pics, lookup)
		if notCached {
			missing = append(missing, file)
		}
		fmt.Fprintln(w, it.Line())
	}
	return missing, in.Err()
}

// icon returns the two-column icon field for it: picture cells when the
// icon of file can be sent, otherwise the glyph it came with, padded. The
// second result is true when the glyph is there only because the icon is
// not cached yet.
func icon(it protocol.Item, file string, pics io.Writer, lookup func(file string) ([]byte, error)) (string, bool) {
	fallback := it.Icon + " "
	if pics == nil {
		return fallback, false
	}
	data, err := lookup(file)
	if err != nil {
		return fallback, errors.Is(err, bundle.ErrNotCached)
	}
	// One id per file, from its path, so the same file always lands in the
	// same slot and a second run replaces rather than piles up.
	id := 1 + int(crc32.ChecksumIEEE([]byte(file))%picture.MaxID)
	if err := picture.Transmit(pics, data, iconCols, iconRows, id); err != nil {
		return fallback, false
	}
	return picture.Cells(id, 0, iconCols), false
}
