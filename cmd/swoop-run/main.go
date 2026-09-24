// swoop-run performs the action for one result, given its id, and exits.
// fzf hands it the id of the picked line with `enter:become(swoop-run {1})`:
// become replaces the fzf process with this one, so there is no shell in
// between and nothing left running when the app is up.
//
// An id that starts with "ext/<name>/" belongs to that extension and is
// handed to it with the prefix removed. Every other id is a built-in
// source's; today that means an application bundle path.
package main

import (
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/ext"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: swoop-run <id>")
		os.Exit(2)
	}
	if err := dispatch(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-run:", err)
		os.Exit(1)
	}
}

func dispatch(id string) error {
	if name, raw, ok := ext.Route(id); ok {
		e, found := ext.Find(name)
		if !found {
			return fmt.Errorf("extension %q is not installed", name)
		}
		return e.Run(raw)
	}
	return run(id)
}
