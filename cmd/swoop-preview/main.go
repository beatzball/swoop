// swoop-preview prints the preview pane for one result, given its id, and
// exits. fzf runs it on every cursor move with `--preview 'swoop-preview
// {1}'`, so it is a hot path: it reads one plist and, after the first time,
// one cached PNG.
//
// The pane size comes from FZF_PREVIEW_COLUMNS and FZF_PREVIEW_LINES, which
// fzf sets for the preview command. Outside fzf a sane default is used.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"strconv"

	"github.com/beatzball/swoop/internal/bundle"
	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/picture"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: swoop-preview <id>")
		os.Exit(2)
	}
	id := os.Args[1]
	cols := envInt("FZF_PREVIEW_COLUMNS", 40)
	rows := envInt("FZF_PREVIEW_LINES", 20)

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	if name, raw, ok := ext.Route(id); ok {
		e, found := ext.Find(name)
		if !found {
			fmt.Fprintf(out, "extension %q is not installed\n", name)
			return
		}
		if err := e.Preview(raw, out); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-preview:", err)
		}
		return
	}

	info := bundle.Describe(id)
	if !picture.Enabled() {
		// The pane starts at the name; a terminal without pictures gets
		// the same details, just no icon above them.
	} else if data, err := info.IconPNG(previewIconPx); err == nil {
		writeIcon(out, data, id, cols, rows)
	} else if !errors.Is(err, bundle.ErrNoIcon) {
		// A broken icon is worth a line on stderr for whoever is debugging,
		// but never worth a blank pane: the details below still print.
		fmt.Fprintln(os.Stderr, "swoop-preview:", err)
	}

	fmt.Fprintf(out, "\x1b[1m%s\x1b[22m\n", info.Name)
	if info.Version != "" {
		fmt.Fprintf(out, "\x1b[2mVersion\x1b[22m  %s\n", info.Version)
	}
	if info.BundleID != "" {
		fmt.Fprintf(out, "\x1b[2mBundle\x1b[22m   %s\n", info.BundleID)
	}
	fmt.Fprintf(out, "\x1b[2mPath\x1b[22m     %s\n", id)
}

// previewIconPx is the icon size for the pane: a 256px icon fills a
// 24-column box on a high-density display without going soft.
const previewIconPx = 256

// iconRows is how much of the pane the icon may take. The details below it
// need four lines; the rest is the picture, up to a size where a 256px
// icon stops getting sharper.
func iconRows(rows int) int {
	r := rows - 5
	if r > 12 {
		r = 12
	}
	if r < 1 {
		r = 1
	}
	return r
}

func writeIcon(out *bufio.Writer, data []byte, id string, cols, rows int) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		cfg = image.Config{Width: 1, Height: 1}
	}
	c, r := picture.Fit(cfg.Width, cfg.Height, cols, iconRows(rows))
	// The image id is derived from the result id so the same app always
	// reuses its slot and the terminal keeps at most 255 icons alive.
	imgID := 1 + int(crc32.ChecksumIEEE([]byte(id))%255)
	if err := picture.WritePNG(out, data, c, r, imgID); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-preview:", err)
		return
	}
	fmt.Fprintln(out)
}

func envInt(name string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return fallback
}
