// swoop-img prints one picture to a terminal that speaks the Kitty graphics
// protocol, sized to fit the fzf preview pane or the given box, and exits.
// It is the thing an extension calls from its own preview command, and the
// quickest way to check that a terminal draws pictures at all:
//
//	swoop-img demo.png
//	swoop-img -cols 30 -rows 10 photo.jpg
//
// PNG is sent as is. Anything else Go can decode (JPEG, GIF) is re-encoded
// as PNG first.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"strconv"

	"github.com/beatzball/swoop/internal/picture"
)

func main() {
	cols := flag.Int("cols", envInt("FZF_PREVIEW_COLUMNS", 40), "width of the box in cells")
	rows := flag.Int("rows", envInt("FZF_PREVIEW_LINES", 20), "height of the box in cells")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: swoop-img [-cols N] [-rows N] <file>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *cols, *rows); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-img:", err)
		os.Exit(1)
	}
}

func run(path string, cols, rows int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	img, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if format != "png" {
		decoded, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, decoded); err != nil {
			return err
		}
		data = buf.Bytes()
	}
	c, r := picture.Fit(img.Width, img.Height, cols, rows)
	id := 1 + int(crc32.ChecksumIEEE([]byte(path))%255)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	return picture.WritePNG(out, data, c, r, id)
}

func envInt(name string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return fallback
}
