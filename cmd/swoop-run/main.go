// swoop-run performs the action for one result, given its id, and exits.
// fzf hands it the id of the picked line with `enter:become(swoop-run {1})`:
// become replaces the fzf process with this one, so there is no shell in
// between and nothing left running when the app is up.
//
// An id that starts with "ext/<name>/" belongs to that extension and is
// handed to it with the prefix removed, with the action if one was picked
// from the menu. Every other id is a built-in source's; today that means
// an application bundle path, which knows open, reveal, and copy-path.
package main

import (
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/ext"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: swoop-run <id> [action]")
		os.Exit(2)
	}
	action := ""
	if len(os.Args) == 3 {
		action = os.Args[2]
	}
	if err := dispatch(os.Args[1], action); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-run:", err)
		os.Exit(1)
	}
}

func dispatch(id, action string) error {
	if name, raw, ok := ext.Route(id); ok {
		e, found := ext.Find(name)
		if !found {
			return fmt.Errorf("extension %q is not installed", name)
		}
		return e.Run(raw, action)
	}
	return run(id, action)
}
