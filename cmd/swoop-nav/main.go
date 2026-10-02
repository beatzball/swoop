// swoop-nav is called by fzf on Enter, Esc, and typing, and prints the
// fzf actions to perform. It keeps the stack of panes in the file named
// by SWOOP_STATE. The rules live in internal/nav; this file only reads
// what fzf hands over and runs the extension for a pane's rows.
//
//	swoop-nav enter [id kind title]   fzf: transform on Enter
//	swoop-nav actions [id kind title] fzf: transform on ctrl-k
//	swoop-nav esc                     fzf: transform on Esc
//	swoop-nav change                  fzf: transform on typing
//	swoop-nav click [id kind title]   fzf: transform on a click on a row
//	swoop-nav key name                fzf: transform on a key an extension claims
//	swoop-nav keys [status]           those keys, for bin/swoop to bind; who has which
//	swoop-nav keys for name           the key that opens extension name's pane
//	swoop-nav landed [kind]           fzf: transform on result-final, once armed
//	swoop-nav step up|down [kind [from [turned]]]
//	                                  fzf: transform after a move of the cursor
//	swoop-nav back id                 fzf: transform after a terminal row's run
//	swoop-nav rows [query]            fzf: reload, prints the current pane
//
// fzf exports FZF_QUERY and FZF_POS to the transform commands, which is
// how the bar's text and the cursor row arrive without any quoting.
package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/match"
	"github.com/beatzball/swoop/internal/nav"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
	"github.com/beatzball/swoop/internal/usage"
)

const envState = "SWOOP_STATE"

// recentCount is how many recent rows lead the root list.
const recentCount = 5

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: swoop-nav enter|actions|click|key|keys|esc|change|landed|step|back|rows|window|divider ...")
		os.Exit(2)
	}
	nav.PreviewPercent = settings.PreviewPercent()
	if os.Args[1] == "window" {
		// bin/swoop's --preview-window at start. No state needed.
		fmt.Println(nav.Window(ext.Pane{}))
		return
	}
	if os.Args[1] == "keys" {
		if len(os.Args) > 3 && os.Args[2] == "for" {
			// For the frame's menu, through the launcher's `keys` verb. No
			// line at all when the extension has no key.
			if k := keyFor(os.Args[3]); k != "" {
				fmt.Println(k)
			}
			return
		}
		keys(len(os.Args) > 2 && os.Args[2] == "status")
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

	// arg is the nth argument after the command, "" when it is not there.
	arg := func(n int) string {
		if len(os.Args) > n+1 {
			return os.Args[n+1]
		}
		return ""
	}

	switch os.Args[1] {
	case "enter", "actions", "click", "key":
		// These read the row under the cursor, or remember its number. A
		// chain's cursor still to be placed is placed first, and the key
		// done again there; see nav.LandEvent. A click is done again as
		// Enter: the cursor is no longer on the row that was clicked.
		again := "swoop-nav " + os.Args[1]
		switch os.Args[1] {
		case "enter", "actions":
			again += " {1} {2} {4}"
		case "click":
			again = "swoop-nav enter {1} {2} {4}"
		case "key":
			// A key's name goes into a shell command as it is, so only
			// the names an extension may claim get that far.
			if !ext.ValidKey(arg(1)) {
				fmt.Println("ignore")
				return
			}
			again += " " + arg(1)
		}
		if acts := nav.Settle(st, query, again); acts != "" {
			fmt.Println(acts)
			if err := nav.Save(path, st); err != nil {
				fmt.Fprintln(os.Stderr, "swoop-nav:", err)
			}
			return
		}
	}

	switch os.Args[1] {
	case "enter", "actions", "click":
		id, kind, title := arg(1), arg(2), arg(3)
		if kind == "group" {
			// A header. A click on it puts the cursor back where it was.
			// A key found it under the cursor before the cursor was moved
			// off it: it is moved now, and the key done again there, once.
			switch {
			case os.Args[1] == "click":
				fmt.Println(nav.Click(st))
				return
			case arg(4) != "again":
				fmt.Println(nav.OffHeader("swoop-nav " + os.Args[1] + " {1} {2} {4} again"))
				return
			}
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
		if nav.Sends(st, id, query) {
			// A pane whose bar is a prompt: send first, and only then
			// clear the bar. A send that fails, because the last answer
			// is still being written, leaves the text where it is.
			if err := send(id, query); err != nil {
				fmt.Println("ignore")
				break
			}
			fmt.Println(nav.AfterSend(st, landed(id), pos))
			break
		}
		var pane ext.Pane
		if kind == "view" {
			pane = paneOf(id)
		}
		fmt.Println(nav.Enter(st, id, kind, title, query, pos, pane, runCommand(path, kind, title)))
	case "key":
		// Among the extensions that are on: one turned off gives its key
		// to the next that claims it, and with none the key does nothing.
		e, claim, ok := ext.ByKey(ext.Discover(ext.Dirs()), arg(1))
		if !ok {
			fmt.Println("ignore")
			return
		}
		fmt.Println(nav.Key(st, ext.Prefix+e.Name+"/"+claim.View, claim.Title, e.Pane(claim.View), query, pos))
	case "divider":
		// +5 gives the preview more, -5 gives the list more. The new
		// width is saved first, so the next run opens the same way.
		delta := 0
		if len(os.Args) > 2 {
			delta, _ = strconv.Atoi(os.Args[2])
		}
		// A pane with a width of its own does not show the user's.
		if nav.OwnWidth(st) {
			fmt.Println("ignore")
			return
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
		fmt.Println(nav.Change(st, keyed(st, query)))
	case "landed":
		// An empty answer is printed as no line at all: fzf takes no
		// action from it.
		if acts := nav.Landed(st, query, arg(1), pos); acts != "" {
			fmt.Println(acts)
		}
	case "step":
		rest := st.Rest
		from, _ := strconv.Atoi(arg(3))
		if acts := nav.Step(st, arg(1), arg(2), pos, from, arg(4) == "turned"); acts != "" {
			fmt.Println(acts)
		}
		// This runs on every Up and Down: the state is written only when
		// the row the cursor rests on has changed.
		if st.Rest == rest {
			return
		}
	case "back":
		ran := ""
		if len(os.Args) > 2 {
			ran = os.Args[2]
		}
		fmt.Println(nav.Back(st, landed(ran)))
	case "rows":
		q := ""
		if len(os.Args) > 2 {
			q = os.Args[2]
		}
		// Untrimmed: "def " with its space is a keyword, "def" is not
		// (see rows); the landing below matches on the trimmed text.
		items, err := rows(st, q)
		q = strings.TrimSpace(q)
		if err != nil {
			fmt.Fprintln(os.Stderr, "swoop-nav:", err)
			os.Exit(1)
		}
		// A landing on a row by id learns its number here, where the rows
		// are; see nav.Landing.
		ids := make([]string, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		if nav.Find(st, q, ids) {
			if err := nav.Save(path, st); err != nil {
				fmt.Fprintln(os.Stderr, "swoop-nav:", err)
			}
		}
		nav.Headers(items)
		if err := protocol.Write(os.Stdout, items); err != nil {
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
func runCommand(statePath, kind, title string) func(target, action string) string {
	return func(target, action string) string {
		// The state file, fzf's socket beside it, the rows kept for the
		// run, and the land file.
		run := "rm -f " + nav.ShellQuote(statePath) + " " + nav.ShellQuote(statePath+".sock")
		for _, env := range []string{envOnce, envLand} {
			if f := os.Getenv(env); f != "" {
				run += " " + nav.ShellQuote(f)
			}
		}
		// The row's kind and title ride along for the usage log. Through
		// env, because the plain terminal's path is "exec …", and exec
		// takes a bare assignment for a command name.
		swoopRun := "env SWOOP_KIND=" + nav.ShellQuote(kind) + " SWOOP_TITLE=" + nav.ShellQuote(title) + " swoop-run " + nav.ShellQuote(target)
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

// envLand names the file a terminal row's run may write a row id to, to
// come back on that row with the bar cleared (nav.Back). bin/swoop sets
// it beside the state file; the run inherits it through fzf.
const envLand = "SWOOP_LAND"

// landed reads and removes what the run of row ran wrote to the land
// file: one id, in the run's own extension's terms, so it gets the same
// prefix as ran. "" when nothing was written.
func landed(ran string) string {
	file := os.Getenv(envLand)
	if file == "" {
		return ""
	}
	data, err := os.ReadFile(file)
	if err != nil {
		// No file is the usual case: the run named no row.
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "swoop-nav:", err)
		}
		return ""
	}
	if err := os.Remove(file); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return ""
	}
	if name, _, ok := ext.Route(ran); ok {
		return ext.Prefix + name + "/" + id
	}
	return id
}

// keys prints the keys the extensions claim, one per line, for bin/swoop
// to bind at start. Every extension's, on or off: one turned on while the
// launcher is open has its key at once, and a key whose extensions are
// all off does nothing (see the key command). With status it prints who
// has which instead, for `<name> status`.
func keys(status bool) {
	if !status {
		for _, k := range ext.Keys(ext.DiscoverAll(ext.Dirs())) {
			fmt.Println(k)
		}
		return
	}
	for _, line := range ext.KeyReport(ext.Discover(ext.Dirs())) {
		fmt.Println("key:     " + line)
	}
}

// keyFor is the key that opens the pane of the extension called name: the
// first key it claims and has, among the extensions that are on. "" when
// it is not installed or is off, claims no key, or another extension, the
// first by name, has every key it claims. So whoever asks gets a key that
// opens that pane and no other, or learns there is none to offer.
func keyFor(name string) string {
	exts := ext.Discover(ext.Dirs())
	for _, e := range exts {
		if e.Name != name {
			continue
		}
		for _, c := range e.Claims() {
			if holder, _, ok := ext.ByKey(exts, c.Key); ok && holder.Name == name {
				return c.Key
			}
		}
	}
	return ""
}

// paneOf is what the view row id's view says about its pane, in its
// extension's views file.
func paneOf(id string) ext.Pane {
	name, raw, ok := ext.Route(id)
	if !ok {
		return ext.Pane{}
	}
	e, found := ext.Find(name)
	if !found {
		return ext.Pane{}
	}
	return e.Pane(raw)
}

// send hands the bar's text to the row id, through its extension.
func send(id, text string) error {
	name, raw, ok := ext.Route(id)
	if !ok {
		return fmt.Errorf("%q is not an extension's row", id)
	}
	e, found := ext.Find(name)
	if !found {
		return fmt.Errorf("extension %q is not installed", name)
	}
	if err := e.Send(raw, text); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
		return err
	}
	return nil
}

// actionsFor is the menu for a row: its extension's answer, or nothing.
func actionsFor(id string) []protocol.Item {
	if name, raw, ok := ext.Route(id); ok {
		if e, found := ext.Find(name); found {
			return e.Actions(raw)
		}
	}
	return nil
}

// rows lists the current pane. At the root that is the rows kept for the
// run plus whatever the other extensions answer for the text (rootRows),
// in one order by title, so a calculator's row for "2+2" sits in the same
// list as the rest; with a keyword first, only that extension's rows (see
// scoped).
// Inside a view it is whatever the view's extension answers. In an actions
// pane it is the target's actions, filtered by the text.
func rows(st *nav.State, query string) ([]protocol.Item, error) {
	top := st.Top()
	if top == nil {
		// Before the trim: "def " is a keyword, "def" is not.
		if e, rest, ok := scope(query); ok {
			return scoped(e, rest)
		}
	}
	query = strings.TrimSpace(query)
	if top == nil {
		items := rootRows(query)
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		})
		// What was opened most recently comes first, marked, unless the
		// setting turns the group off.
		if settings.Get(settings.Recent, settings.RecentDefault) != "off" {
			if entries, err := usage.Load(); err == nil {
				items = usage.Front(items, usage.Recent(entries, recentCount), recentCount)
			}
		}
		return items, nil
	}
	if top.Kind == "actions" {
		q := match.New(query)
		return match.Rank(actionsFor(top.View), func(it protocol.Item) (int, bool) { return q.Score(it.Title) }), nil
	}
	name, viewID, ok := ext.Route(top.View)
	if !ok {
		return nil, fmt.Errorf("%q is not an extension view", top.View)
	}
	e, found := ext.Find(name)
	if !found {
		return nil, fmt.Errorf("extension %q is not installed", name)
	}
	if top.Pane.Prompt {
		// The bar there is the prompt being written, not a filter.
		query = ""
	}
	return e.View(viewID, query)
}

// scope finds the extension a keyword at the start of the root bar names,
// and the text after it. Nothing is read from the disk for a bar without
// a space in it: most keystrokes.
func scope(query string) (ext.Extension, string, bool) {
	kw, rest, ok := ext.SplitKeyword(query)
	if !ok {
		return ext.Extension{}, "", false
	}
	e, found := ext.ByKeyword(ext.Discover(ext.Dirs()), kw)
	return e, rest, found
}

// keyed is what Change needs to know about a keyword at the root: the
// text after it, and the view the keyword opens: the one the extension's
// key opens, when it claims one, since such an extension may list nothing
// at the root; else its view row when its list is that one row.
func keyed(st *nav.State, query string) *nav.Keyed {
	if st.Top() != nil {
		return nil
	}
	e, rest, ok := scope(query)
	if !ok {
		return nil
	}
	k := &nav.Keyed{Rest: rest}
	for _, c := range e.Claims() {
		if ext.ValidKey(c.Key) {
			k.View, k.Title, k.Pane = ext.Prefix+e.Name+"/"+c.View, c.Title, e.Pane(c.View)
			return k
		}
	}
	if items, err := e.List(""); err == nil && len(items) == 1 && items[0].Kind == "view" {
		k.View, k.Title, k.Pane = items[0].ID, items[0].Title, paneOf(items[0].ID)
	}
	return k
}

// scoped prints the root scoped to one extension by its keyword: its rows
// for the rest of the bar, and only those that match every word of it,
// best first. fzf's matching is off here (see nav.Change), because the bar still
// starts with the keyword, so the filtering is done here, the way the
// actions pane does it. No other rows and no recent group: the keyword
// asked for one extension.
func scoped(e ext.Extension, rest string) ([]protocol.Item, error) {
	items, err := e.List(strings.TrimSpace(rest))
	if err != nil {
		return nil, err
	}
	return matching(items, rest), nil
}

// matching keeps the items that match every word of query in the title
// or the subtitle, best first, by the rules every view filters with
// (internal/match).
func matching(items []protocol.Item, query string) []protocol.Item {
	return match.Items(match.New(query), items)
}
