// swoop-nav is called by fzf on Enter, Esc, and typing, and prints the
// fzf actions to perform. It keeps the stack of panes in the file named
// by SWOOP_STATE. The rules live in internal/nav; this file only reads
// what fzf hands over and runs the extension for a pane's rows.
//
//	swoop-nav enter [id kind title]   fzf: transform on Enter
//	swoop-nav esc                     fzf: transform on Esc
//	swoop-nav change                  fzf: transform on typing, in a view
//	swoop-nav rows [query]            fzf: reload, prints the current pane
//
// fzf exports FZF_QUERY and FZF_POS to the transform commands, which is
// how the bar's text and the cursor row arrive without any quoting.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/nav"
	"github.com/beatzball/swoop/internal/protocol"
)

const envState = "SWOOP_STATE"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: swoop-nav enter|esc|change|rows ...")
		os.Exit(2)
	}
	path := os.Getenv(envState)
	if path == "" {
		fmt.Fprintln(os.Stderr, "swoop-nav: "+envState+" is not set")
		os.Exit(2)
	}
	st, err := nav.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
		// A broken state file must not trap the user: act as the root.
		st = &nav.State{}
	}

	query := os.Getenv("FZF_QUERY")
	pos, _ := strconv.Atoi(os.Getenv("FZF_POS"))

	switch os.Args[1] {
	case "enter":
		var id, kind, title string
		if len(os.Args) > 2 {
			id = os.Args[2]
		}
		if len(os.Args) > 3 {
			kind = os.Args[3]
		}
		if len(os.Args) > 4 {
			title = os.Args[4]
		}
		// The run command removes the state file itself: become replaces
		// fzf, so nothing after it in bin/swoop ever runs.
		run := "rm -f " + nav.ShellQuote(path) + "; exec swoop-run " + nav.ShellQuote(id)
		fmt.Println(nav.Enter(st, id, kind, title, query, pos, run))
	case "esc":
		fmt.Println(nav.Esc(st, query))
	case "change":
		fmt.Println(nav.Change(st))
	case "rows":
		q := ""
		if len(os.Args) > 2 {
			q = os.Args[2]
		}
		if err := rows(st, q); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-nav:", err)
			os.Exit(1)
		}
		return
	default:
		fmt.Fprintln(os.Stderr, "swoop-nav: unknown command", os.Args[1])
		os.Exit(2)
	}
	if err := nav.Save(path, st); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
	}
}

// rows prints the current pane. At the root that is the same pipeline
// bin/swoop ran at startup; the pictures are already in the terminal, so
// swoop-icons only rewrites the rows and its picture output is dropped.
func rows(st *nav.State, query string) error {
	top := st.Top()
	if top == nil {
		list := exec.Command("swoop-list")
		icons := exec.Command("swoop-icons", "-out", os.DevNull)
		pipe, err := list.StdoutPipe()
		if err != nil {
			return err
		}
		icons.Stdin = pipe
		icons.Stdout = os.Stdout
		icons.Stderr = os.Stderr
		list.Stderr = os.Stderr
		if err := list.Start(); err != nil {
			return err
		}
		if err := icons.Run(); err != nil {
			return err
		}
		return list.Wait()
	}
	name, viewID, ok := ext.Route(top.View)
	if !ok {
		return fmt.Errorf("%q is not an extension view", top.View)
	}
	e, found := ext.Find(name)
	if !found {
		return fmt.Errorf("extension %q is not installed", name)
	}
	items, err := e.View(viewID, strings.TrimSpace(query))
	if err != nil {
		return err
	}
	return protocol.Write(os.Stdout, items)
}
