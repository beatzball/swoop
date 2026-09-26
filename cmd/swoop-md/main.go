// swoop-md renders markdown as styled text for a terminal: markdown on
// stdin or from a file, text with ANSI styles on stdout.
//
//	swoop-md [-w WIDTH] [FILE]
//
// The width is -w, else FZF_PREVIEW_COLUMNS, else COLUMNS, else 80. It is
// internal/markdown behind a pipe, the same renderer the Ask AI pane
// uses, so anything else can use it too: a preview command, a script, a
// tool of your own. No highlighter, no theme file, one binary.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/beatzball/swoop/internal/markdown"
)

func main() {
	width := flag.Int("w", 0, "wrap at this many columns (default: FZF_PREVIEW_COLUMNS, COLUMNS, or 80)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: swoop-md [-w WIDTH] [FILE]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	in := os.Stdin
	if flag.NArg() == 1 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "swoop-md:", err)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}
	data, err := io.ReadAll(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-md:", err)
		os.Exit(1)
	}
	fmt.Print(markdown.Render(string(data), widthOr(*width)))
}

// widthOr is the width to use: the flag when given, else what the
// terminal or fzf says, else 80.
func widthOr(flagged int) int {
	if flagged > 0 {
		return flagged
	}
	for _, name := range []string{"FZF_PREVIEW_COLUMNS", "COLUMNS"} {
		if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
			return v
		}
	}
	return 80
}
