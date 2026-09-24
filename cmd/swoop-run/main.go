// swoop-run performs the action for one result, given its id, and exits.
// fzf hands it the id of the picked line with `enter:become(swoop-run {1})`:
// become replaces the fzf process with this one, so there is no shell in
// between and nothing left running when the app is up.
//
// Today every id is an application bundle path. When more kinds exist, the
// kind will have to travel with the id; that is an open question on the
// protocol spec, not something to guess at here.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: swoop-run <id>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-run:", err)
		os.Exit(1)
	}
}
