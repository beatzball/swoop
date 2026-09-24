// swoop-list prints every result the built-in sources know about, one
// protocol line per result, and exits. It is the hot path: fzf may run it
// on keystrokes, so it does nothing but list and print.
//
// A query argument is accepted and ignored. The apps source is static, and
// fzf does the matching; a dynamic source (calculator, file search) will be
// the first to read it.
package main

import (
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/apps"
	"github.com/beatzball/swoop/internal/protocol"
)

func main() {
	items, err := apps.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-list:", err)
		os.Exit(1)
	}
	if err := protocol.Write(os.Stdout, items); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-list:", err)
		os.Exit(1)
	}
}
