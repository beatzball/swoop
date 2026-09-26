// swoop-nav is called by fzf on Enter, Esc, and typing, and prints the
// fzf actions to perform. It keeps the stack of panes in the file named
// by SWOOP_STATE. The rules live in internal/nav; this file only reads
// what fzf hands over and runs the extension for a pane's rows.
//
//	swoop-nav enter [id kind title]   fzf: transform on Enter
//	swoop-nav actions [id kind title] fzf: transform on ctrl-k
//	swoop-nav esc                     fzf: transform on Esc
//	swoop-nav change                  fzf: transform on typing
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

	"github.com/beatzball/swoop/internal/apps"
	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/nav"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
)

const envState = "SWOOP_STATE"

// envApps names the file bin/swoop filled at startup with the built-in
// rows, pictures included. Root reloads read it instead of listing apps
// again: they do not change while the launcher is open, and a reload
// happens on every keystroke.
const envApps = "SWOOP_APPS"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: swoop-nav enter|actions|ai|esc|change|rows|window|divider ...")
		os.Exit(2)
	}
	nav.PreviewPercent = settings.PreviewPercent()
	if os.Args[1] == "window" {
		// bin/swoop's --preview-window at start. No state needed.
		fmt.Println(nav.Window(false))
		return
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
	case "enter", "actions":
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
		if os.Args[1] == "actions" {
			// A row with nothing to offer: ctrl-k does nothing.
			if id == "" || len(actionsFor(id)) == 0 {
				fmt.Println("ignore")
				return
			}
			fmt.Println(nav.Actions(st, id, kind, title, query, pos))
			break
		}
		if target, ok := nav.AISendTarget(st, id, query); ok {
			// The Ask AI pane: send first, and only then clear the bar.
			// A send that fails, because the last answer is still being
			// written, leaves the text where it is.
			send := exec.Command("swoop-ai", "send", target, query)
			send.Stderr = os.Stderr
			if err := send.Run(); err != nil {
				fmt.Println("ignore")
				break
			}
			fmt.Println(nav.AfterSend())
			break
		}
		fmt.Println(nav.Enter(st, id, kind, title, query, pos, runCommand(path)))
	case "ai":
		fmt.Println(nav.Ask(st, query, pos))
	case "divider":
		// +5 gives the preview more, -5 gives the list more. The new
		// width is saved first, so the next run opens the same way.
		delta := 0
		if len(os.Args) > 2 {
			delta, _ = strconv.Atoi(os.Args[2])
		}
		nav.PreviewPercent = settings.ClampPreview(nav.PreviewPercent + delta)
		if err := settings.Set(settings.Preview, strconv.Itoa(nav.PreviewPercent)); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-nav:", err)
		}
		fmt.Println(nav.Divider(st))
		return
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

// runCommand builds the fzf action that performs a row and ends the
// launcher: remove the run's files, run the action, and in a frame of our
// own tell the frame the launcher is leaving. In a plain terminal it is
// become, which replaces fzf with the command, so nothing after fzf in
// bin/swoop ever runs; that is why the files go here. The colon form
// takes the rest of the string, which keeps any character in the command
// safe.
func runCommand(statePath string) func(target, action string) string {
	return func(target, action string) string {
		// The state file, fzf's socket beside it, and the apps cache.
		run := "rm -f " + nav.ShellQuote(statePath) + " " + nav.ShellQuote(statePath+".sock")
		if apps := os.Getenv(envApps); apps != "" {
			run += " " + nav.ShellQuote(apps)
		}
		swoopRun := "swoop-run " + nav.ShellQuote(target)
		if action != "" {
			swoopRun += " " + nav.ShellQuote(action)
		}
		if shell := os.Getenv("SWOOP_SHELL_PID"); shell != "" {
			// Inside a frame of our own: run the action, THEN tell the frame
			// the launcher is leaving. The frame answers the signal by
			// dropping the surface, which ends everything in it, so a
			// signal sent first would kill the runner before it ran.
			//
			// execute-silent rather than become: become restores the
			// terminal's primary screen first, and the runner takes 40 ms
			// or more, so the panel showed the shell's "Last login" line
			// until the signal landed. execute-silent keeps fzf on its own
			// screen while the command runs; the frame drops the surface
			// before abort is ever reached, and nothing else is seen.
			return nav.Wrap("execute-silent", run+"; "+swoopRun+"; kill -USR2 "+nav.ShellQuote(shell)+" 2>/dev/null") + "+abort"
		}
		return "become:" + run + "; exec " + swoopRun
	}
}

// actionsFor is the menu for a row: the launcher's own three for an app,
// the extension's answer for one of its rows, nothing for the rest.
func actionsFor(id string) []protocol.Item {
	if name, raw, ok := ext.Route(id); ok {
		if e, found := ext.Find(name); found {
			return e.Actions(raw)
		}
		return nil
	}
	if strings.HasSuffix(id, ".app") {
		return apps.Actions()
	}
	return nil
}

// rows prints the current pane. At the root that is the cached built-in
// rows plus whatever the extensions answer for the text, in one order by
// title, so a calculator's row for "2+2" sits in the same list as the apps.
// Inside a view it is whatever the view's extension answers. In an actions
// pane it is the target's actions, filtered by the text.
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
	if top.Kind == "actions" {
		var items []protocol.Item
		for _, it := range actionsFor(top.View) {
			if query == "" || strings.Contains(strings.ToLower(it.Title), strings.ToLower(query)) {
				items = append(items, it)
			}
		}
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
	if top.Kind == "ai" {
		// The bar there is the prompt being written, not a filter.
		query = ""
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
