// swoop-run performs the action for one result, given its id, and exits.
// fzf hands it the id of the picked line with `enter:become(swoop-run {1})`:
// become replaces the fzf process with this one, so there is no shell in
// between and nothing left running when the app is up.
//
// An id that starts with "ext/<name>/" belongs to that extension and is
// handed to it with the prefix removed, with the action if one was picked
// from the menu. Every row is an extension's, so any other id is an error.
package main

import (
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/usage"
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
	// An open, counted: Enter's default, or Open from the action menu.
	// Enter on a terminal row counts too, with its kind and title: an
	// edit of a note is an open of it. A refresh or terminal action
	// (delete, a setting's choice, Edit the list) is not an open and is
	// not counted. bin/swoop hands the row's kind and
	// title over in the environment; swoop-run by hand has neither.
	if action == "" || action == "open" {
		_ = usage.Record(os.Args[1], os.Getenv("SWOOP_KIND"), os.Getenv("SWOOP_TITLE"))
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
	return fmt.Errorf("%q is not an extension's row", id)
}
