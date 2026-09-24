// swoop-list prints every result the built-in sources and the installed
// extensions know about, one protocol line per result, and exits. It is
// the hot path, so it does nothing but list, merge, sort, and print.
//
// A query argument is accepted and passed to extensions; the apps source
// is static and ignores it, and fzf does the matching.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/beatzball/swoop/internal/apps"
	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/protocol"
)

func main() {
	query := ""
	if len(os.Args) > 1 {
		query = os.Args[1]
	}
	items, err := apps.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-list:", err)
		os.Exit(1)
	}
	// Every extension's list runs at the same time, inside ListAll. The
	// apps source is not overlapped with them: it takes two milliseconds.
	items = append(items, ext.ListAll(ext.Discover(ext.Dirs()), query)...)
	// One list, one order: by title, without regard to case, so an
	// extension's rows sit among the apps rather than under them.
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
	if err := protocol.Write(os.Stdout, items); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-list:", err)
		os.Exit(1)
	}
}
