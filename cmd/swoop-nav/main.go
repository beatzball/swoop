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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/nav"
	"github.com/beatzball/swoop/internal/protocol"
)

const envState = "SWOOP_STATE"

// envApps names the file bin/swoop filled at startup with the built-in
// rows, pictures included. Root reloads read it instead of listing apps
// again: they do not change while the launcher is open, and a reload
// happens on every keystroke.
const envApps = "SWOOP_APPS"

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
		// The run command removes the run's files itself: become replaces
		// fzf, so nothing after it in bin/swoop ever runs.
		run := "rm -f " + nav.ShellQuote(path)
		if apps := os.Getenv(envApps); apps != "" {
			run += " " + nav.ShellQuote(apps)
		}
		// Inside a frame of our own, tell it the launcher is leaving; see
		// the same line in bin/swoop.
		if shell := os.Getenv("SWOOP_SHELL_PID"); shell != "" {
			run += "; kill -USR2 " + nav.ShellQuote(shell) + " 2>/dev/null"
		}
		run += "; exec swoop-run " + nav.ShellQuote(id)
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

// rows prints the current pane. At the root that is the cached built-in
// rows plus whatever the extensions answer for the text, in one order by
// title, so a calculator's row for "2+2" sits in the same list as the apps.
// Inside a view it is whatever the view's extension answers.
func rows(st *nav.State, query string) error {
	query = strings.TrimSpace(query)
	top := st.Top()
	if top == nil {
		items, err := cachedApps()
		if err != nil {
			return err
		}
		items = append(items, ext.ListAll(ext.Discover(ext.Dirs()), query)...)
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		})
		return protocol.Write(os.Stdout, items)
	}
	name, viewID, ok := ext.Route(top.View)
	if !ok {
		return fmt.Errorf("%q is not an extension view", top.View)
	}
	e, found := ext.Find(name)
	if !found {
		return fmt.Errorf("extension %q is not installed", name)
	}
	items, err := e.View(viewID, query)
	if err != nil {
		return err
	}
	return protocol.Write(os.Stdout, items)
}

// cachedApps reads the rows bin/swoop cached at startup. Without the cache
// (swoop-nav run by hand) it lists the apps directly, without pictures.
func cachedApps() ([]protocol.Item, error) {
	path := os.Getenv(envApps)
	var data []byte
	var err error
	if path != "" {
		data, err = os.ReadFile(path)
	} else {
		data, err = exec.Command("swoop-list").Output()
	}
	if err != nil {
		return nil, err
	}
	var items []protocol.Item
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		if it, err := protocol.Parse(sc.Text()); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}
