// swoop-preview prints the preview pane for one result, given its id, and
// exits. fzf runs it on every cursor move with `--preview 'swoop-preview
// {1}'`, so it is a hot path: it finds the extension the id belongs to
// and hands over to its `preview`, with the prefix removed.
//
// The pane size is in FZF_PREVIEW_COLUMNS and FZF_PREVIEW_LINES, which
// fzf sets for the preview command and the extension inherits.
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/ext"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: swoop-preview <id>")
		os.Exit(2)
	}
	id := os.Args[1]

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

	// Every row is an extension's. An id without the prefix is nobody's.
	fmt.Fprintf(out, "%q is not an extension's row\n", id)
}
