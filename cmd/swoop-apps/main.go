// swoop-apps is the applications on this machine, as an extension: the
// executable behind extensions/apps. It speaks the extension contract and
// nothing more, so the launcher routes to it like any other.
//
//	swoop-apps list                  every application, one line each
//	swoop-apps preview <id>          the icon, name, version and path
//	swoop-apps actions <id>          Open, reveal, Copy path
//	swoop-apps run <id> [action]     open it, or the action picked
//
// The id is the application's path: the .app bundle on macOS, the .desktop
// file on Linux. The list does not depend on the text in the bar and does
// not change while the launcher is open, so the extension asks, with the
// files "cache" and "icons" beside it, to be listed once per launch and to
// have each row's icon taken from the file its id names. See internal/ext.
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/apps"
	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/usage"
)

// name is the extension's name, the directory this tool is run from.
const name = "apps"

func main() {
	arg := func(i int) string {
		if len(os.Args) > i {
			return os.Args[i]
		}
		return ""
	}
	var err error
	switch arg(1) {
	case "list":
		// A query is accepted and ignored: the list is the same for any
		// text, and fzf does the matching.
		err = list()
	case "preview":
		out := bufio.NewWriter(os.Stdout)
		preview(out, arg(2))
		err = out.Flush()
	case "actions":
		if arg(2) != "" {
			err = protocol.Write(os.Stdout, apps.Actions())
		}
	case "run":
		if arg(2) == "" {
			usageExit()
		}
		err = run(arg(2), arg(3))
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-apps:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-apps list | preview <id> | actions <id> | run <id> [action]")
	os.Exit(2)
}

func list() error {
	items, err := apps.List()
	if err != nil {
		return err
	}
	if err := protocol.Write(os.Stdout, items); err != nil {
		return err
	}
	// The apps were the launcher's own rows once, and an open of one was
	// logged under its bare path. Those opens are this extension's now, so
	// Used recently and Stats go on counting the same apps. After the
	// first launch there is nothing left to take, and this is one read of
	// the log. A log that cannot be rewritten costs the old counts only.
	_ = usage.Adopt(ext.Prefix + name + "/")
	return nil
}
