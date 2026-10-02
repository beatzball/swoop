// swoop-window is the window extension: rows that move and size the
// focused window of the app in front. Halves, thirds, quarters, maximize,
// almost maximize, reasonable size, center, and the next or previous
// display.
//
//	swoop-window list           the root rows, one per command
//	swoop-window preview <id>   the window it will move and where to
//	swoop-window run <id>       move it
//
// The launcher's panel does not take focus from the app you were in, so
// that app's window is the one a row moves. On macOS it goes through the
// Accessibility API, which needs the permission; without it, run opens
// the settings pane and the preview says what to switch on. The
// arithmetic is in geometry.go, apart from the calls, so it is tested on
// every OS.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/tool"
)

// icon is a window in the Nerd Font: fa-window_restore.
const icon = ""

// errNotTrusted is what a call says without the permission. It names the
// app to switch on: macOS asks of the app that started this process, the
// frame, or a terminal when the launcher runs in one. A switch that shows
// on can still be stale, a grant to an earlier build, and this check
// cannot tell the two apart, so the message covers both.
var errNotTrusted = errors.New("moving windows needs Accessibility: in System Settings, Privacy & Security, Accessibility, turn on swoop-shell-mac (or the terminal " + tool.Read().Title + " runs in), then try again. Already on? Remove it with the minus button and add it again: a new build needs a new grant")

func main() {
	if len(os.Args) < 2 {
		usageExit()
	}
	id := ""
	if len(os.Args) > 2 {
		id = os.Args[2]
	}
	var err error
	switch os.Args[1] {
	case "list":
		err = list()
	case "preview":
		err = preview(id)
	case "run":
		err = run(id)
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-window:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-window list | preview <id> | run <id>")
	os.Exit(2)
}

// list prints every command. fzf filters at the root, so what was typed
// does not matter here.
func list() error {
	items := make([]protocol.Item, len(commands))
	for i, c := range commands {
		items[i] = protocol.Item{ID: c.ID, Kind: "command", Icon: icon, Title: c.Title, Subtitle: c.Subtitle}
	}
	return protocol.Write(os.Stdout, items)
}

// preview says which window the row will move, where it is, and where it
// will go. Without the permission it says what to switch on instead,
// before Enter is ever pressed. It never asks for the permission itself:
// the preview runs on every cursor move.
func preview(id string) error {
	c, ok := findCommand(id)
	if !ok {
		return fmt.Errorf("no window command %q", id)
	}
	fmt.Println(c.Title)
	fmt.Println()
	fmt.Println(c.Subtitle)
	fmt.Println()
	if !trusted() {
		fmt.Println("Needs Accessibility. macOS lets an app move other apps'")
		fmt.Println("windows only once you switch it on.")
		fmt.Println()
		fmt.Println("Enter opens System Settings, Privacy & Security,")
		fmt.Println("Accessibility. Turn on swoop-shell-mac (or the terminal")
		fmt.Println(tool.Read().Title + " runs in), then try again.")
		fmt.Println()
		fmt.Println("Already on? Remove it with the minus button and add it")
		fmt.Println("again: a new build needs a new grant.")
		return nil
	}
	w, to, d, n, err := plan(id)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	name := w.App
	if w.Title != "" {
		name += " · " + w.Title
	}
	fmt.Println("Window:  ", name)
	fmt.Println("Now:     ", w.Frame)
	fmt.Println("Then:    ", to)
	fmt.Printf("Display:  %d of %d\n", d+1, n)
	return nil
}

// plan is the window in front and the frame the command gives it, with
// the display it lands on and how many there are.
func plan(id string) (Window, Rect, int, int, error) {
	w, err := focused()
	if err != nil {
		return Window{}, Rect{}, 0, 0, err
	}
	ds, err := displays()
	if err != nil {
		return Window{}, Rect{}, 0, 0, err
	}
	to, d, err := Place(id, w.Frame, ds)
	return w, to, d, len(ds), err
}

// run moves the window. The first run without the permission explains on
// stderr and opens the pane where it is granted, rather than failing
// with nothing to show for it.
func run(id string) error {
	if _, ok := findCommand(id); !ok {
		return fmt.Errorf("no window command %q", id)
	}
	if !trusted() {
		if err := askTrust(); err != nil {
			return err
		}
		return errNotTrusted
	}
	w, to, _, _, err := plan(id)
	if err != nil {
		return err
	}
	return move(w, to)
}
