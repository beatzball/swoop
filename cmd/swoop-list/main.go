// swoop-list prints every result the built-in sources know about, one
// protocol line per result, and exits. Today that is the apps. It is run
// once per launch: the rows are cached for the run, because apps do not
// change while the launcher is open, and swoop-nav merges the extensions'
// rows in on every keystroke.
//
// A query argument is accepted and ignored: the apps source is static.
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
