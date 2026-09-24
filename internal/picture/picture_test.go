package picture

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
)

func TestFitKeepsShape(t *testing.T) {
	cases := []struct {
		w, h, maxCols, maxRows, cols, rows int
	}{
		{128, 128, 40, 8, 16, 8}, // square: rows win, cols = 2*rows
		{128, 128, 10, 8, 10, 5}, // square but narrow pane: cols win
		{400, 100, 40, 8, 40, 5}, // wide picture: cols win
		{100, 400, 40, 8, 4, 8},  // tall picture: rows win
		{0, 0, 40, 8, 1, 1},      // nonsense sizes still give a box
		{128, 128, 0, 0, 1, 1},   // no room still gives a box
	}
	for _, c := range cases {
		cols, rows := Fit(c.w, c.h, c.maxCols, c.maxRows)
		if cols != c.cols || rows != c.rows {
			t.Errorf("Fit(%d,%d,%d,%d) = %d,%d; want %d,%d", c.w, c.h, c.maxCols, c.maxRows, cols, rows, c.cols, c.rows)
		}
	}
}

// testPNG makes a size-by-size PNG of random pixels. Random, not patterned:
// PNG compresses a pattern to almost nothing, and the chunking test needs
// an image that stays big enough to need several chunks.
func testPNG(t *testing.T, size int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	rng := rand.New(rand.NewSource(1))
	for i := range img.Pix {
		img.Pix[i] = byte(rng.Intn(256))
	}
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestWritePNGShape(t *testing.T) {
	var out bytes.Buffer
	cols, rows := 6, 3
	if err := WritePNG(&out, testPNG(t, 8), cols, rows, 7); err != nil {
		t.Fatal(err)
	}
	s := out.String()

	// One transmit escape, carrying the options and the whole (small) image.
	if !strings.HasPrefix(s, "\x1b_Ga=T,f=100,i=7,c=6,r=3,U=1,q=2,m=0;") {
		t.Fatalf("transmit options wrong: %q", s[:60])
	}
	if strings.Count(s, "\x1b_G") != 1 {
		t.Fatalf("want one chunk for a tiny image, got %d", strings.Count(s, "\x1b_G"))
	}

	// rows lines of placeholders, each cols cells wide, each ending in a
	// colour reset and a newline so fzf sees plain lines.
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	if len(lines) != rows {
		t.Fatalf("want %d lines, got %d", rows, len(lines))
	}
	for i, l := range lines {
		if n := strings.Count(l, string(kitty.Placeholder)); n != cols {
			t.Errorf("line %d: want %d placeholder cells, got %d", i, cols, n)
		}
		if !strings.HasPrefix(l[strings.Index(l, "\x1b["):], "\x1b[38;5;7m") && !strings.Contains(l, "\x1b[38;5;7m") {
			t.Errorf("line %d: image id not set in the foreground colour", i)
		}
		if !strings.HasSuffix(l, "\x1b[39m") {
			t.Errorf("line %d: does not end with a colour reset", i)
		}
	}
	if s[len(s)-1] != '\n' {
		t.Fatal("output must end with a newline")
	}
}

func TestWritePNGChunks(t *testing.T) {
	var out bytes.Buffer
	if err := WritePNG(&out, testPNG(t, 96), 4, 2, 1); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	chunks := strings.Count(s, "\x1b_G")
	if chunks < 2 {
		t.Fatalf("a 96px noisy PNG should need more than one chunk, got %d", chunks)
	}
	if strings.Count(s, ",m=1;")+strings.Count(s, "\x1b_Gm=1;") != chunks-1 {
		t.Fatalf("every chunk but the last must say m=1: %d chunks", chunks)
	}
	if !strings.Contains(s, "\x1b_Gm=0;") {
		t.Fatal("the last chunk must say m=0")
	}
}

func TestWritePNGRejectsBadArgs(t *testing.T) {
	var out bytes.Buffer
	if err := WritePNG(&out, testPNG(t, 4), 1, 1, 0); err == nil {
		t.Error("id 0 accepted")
	}
	if err := WritePNG(&out, testPNG(t, 4), 1, 1, 256); err == nil {
		t.Error("id 256 accepted")
	}
	if err := WritePNG(&out, testPNG(t, 4), 0, 1, 1); err == nil {
		t.Error("0 columns accepted")
	}
	if err := WritePNG(&out, testPNG(t, 4), MaxCells+1, 1, 1); err == nil {
		t.Error("too many columns accepted")
	}
}
