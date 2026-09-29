// Package nav is the launcher's memory of where the user is: a stack of
// panes. fzf itself only ever shows one list; every "go into Define Word",
// every "show this row's actions", and every "Esc back out" is fzf swapping
// that list on the instructions this package prints. The rules, from the
// navigation and action-menu design issues:
//
//   - Esc with text in the bar clears the text. Esc with an empty bar
//     closes at the root, or pops one pane inside a view.
//   - Enter on a row of kind "view" pushes a pane: the view's rows, its
//     own prompt, an empty bar, and fzf's own matching turned off so the
//     extension does the filtering. Enter on any other row runs it.
//   - ctrl-k on a row pushes an actions pane for it. Enter on an action of
//     kind "action" runs it and ends the launcher; kind "refresh" runs it
//     and returns to the pane it came from, reloaded.
//   - Enter on a row of kind "toggle" inside a view runs it and stays: the
//     same pane, the bar cleared, reloaded, the cursor on the same row
//     number. A checklist is ticked off one row after another; "refresh"
//     would leave the pane after each one. Two quick Enters tick two
//     rows: no key is dropped while the list reloads (see LandEvent).
//   - Enter on a row of kind "terminal" hands the whole terminal to its
//     run, an editor on a note, and takes it back when the run exits:
//     the same pane, the bar as it was, reloaded, the cursor on the same
//     row number, the preview redrawn. fzf's preview is read-only; this
//     is how anything that needs a cursor happens inside the panel. An
//     action of kind "terminal" does the same from the actions pane,
//     then returns to the pane below, like "refresh".
//   - A terminal row's run may name a row to come back to, by writing its
//     id to the file in $SWOOP_LAND. Then the bar comes back empty and the
//     cursor on that row: New note returns on the note it made, and a
//     second Enter does not make a second one. See Back.
//   - Popping restores the text and the cursor row the user left.
//   - Tab opens the Ask AI pane from anywhere, and keeps the bar's text.
//     The bar there is a prompt, not a filter: Enter sends it to the
//     conversation under the cursor, or to a new one, and the answer
//     arrives on the right. Tab inside the pane does nothing.
//   - A keyword and a space at the start of the root bar scope it to one
//     extension: "def ap" opens Define with "ap" typed, "win l" shows
//     only Window's rows for "l". See Change.
//
// The functions here return fzf action strings. They do no I/O of their
// own except through Load and Save, so they can be tested without fzf.
package nav

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Frame is one pushed pane.
type Frame struct {
	// Kind is "view" for an extension's pane, "actions" for a row's
	// actions, "ai" for the pane Tab opens.
	Kind string `json:"kind"`
	// View is the id of the row that opened it: the view row for a view,
	// the target row for an actions pane.
	View  string `json:"view"`
	Title string `json:"title"` // its title, used as the prompt
	Query string `json:"query"` // the bar's text at the moment of Enter; the question, for "ai"
	Pos   int    `json:"pos"`   // the cursor row at the moment of Enter, 1-based
}

// State is the stack. Empty means the root list.
type State struct {
	Stack []Frame `json:"stack"`
	// Land is where the cursor goes once the reload in flight lands, or
	// nil. See LandEvent.
	Land *Landing `json:"land,omitempty"`
}

// Landing is where a chain wants the cursor once its reload lands on a
// list other than the one on screen: the row a pop returns to, the
// conversation a send made, the note a New note made.
type Landing struct {
	// Query is the bar's text the chain left. Text typed since means the
	// user has moved on, and the cursor stays where the typing put it.
	Query string `json:"query"`
	Pos   int    `json:"pos"` // the row, 1-based
	// ID names the row instead of Pos. Its number is known only once the
	// rows are, so swoop-nav rows fills in Pos as it prints them (Find).
	ID      string `json:"id,omitempty"`
	Refresh bool   `json:"refresh,omitempty"` // redraw the preview once there
}

// Find fills in the row number of a Landing by id from the rows a
// reload for query prints, in order. It says whether it did, so the
// caller saves the state only then.
func Find(st *State, query string, ids []string) bool {
	l := st.Land
	if l == nil || l.ID == "" || l.Pos != 0 || l.Query != query {
		return false
	}
	for i, id := range ids {
		if id == l.ID {
			l.Pos = i + 1
			return true
		}
	}
	return false
}

// Top is the pane the user is in, or nil at the root.
func (s *State) Top() *Frame {
	if len(s.Stack) == 0 {
		return nil
	}
	return &s.Stack[len(s.Stack)-1]
}

// Load reads the state file. A missing file is the root, not an error:
// the file is created on the first push.
func Load(path string) (*State, error) {
	st := &State{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return st, nil
	}
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("nav: state file: %w", err)
	}
	return st, nil
}

// Save writes the state file. Whole or not at all, through a rename:
// swoop-nav rows saves it too (Find), during a reload, while a key's
// transform may be reading it, and half a file reads as the root.
func Save(path string, st *State) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	// A name of its own, as two savers can be at it at once.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

// RootPrompt is the bar's prompt at the root.
const RootPrompt = "  "

// Enter decides what Enter does for the current row. id, kind, and title
// are the row's fields; query and pos are the bar's text and the cursor
// row at that moment. A view row pushes a pane; an action row runs its
// action; any other row becomes the run command; no row at all is
// ignored. runCmd turns a target id and an action into the fzf action
// that performs it and ends the launcher, prefix and all: become in a
// plain terminal, execute-silent and abort inside a frame (see
// cmd/swoop-nav). "" action means the default.
func Enter(st *State, id, kind, title, query string, pos int, runCmd func(target, action string) string) string {
	if id == "" {
		return "ignore"
	}
	if top := st.Top(); top != nil && top.Kind == "actions" {
		return enterAction(st, top, id, kind, runCmd)
	}
	if top := st.Top(); top != nil && top.Kind == "ai" {
		// The caller sends first; see AISendTarget. Enter here never
		// runs a row or pushes a pane.
		return "ignore"
	}
	if kind == "refresh" && st.Top() != nil {
		// A row that changes something and stays: a setting's choice.
		// Run it quietly, then back to the pane below, reloaded, so it
		// shows the change. The same shape as a refresh action.
		return Wrap("execute-silent", "swoop-run "+ShellQuote(id)) + "+" + popActions(st, false)
	}
	if kind == "toggle" && st.Top() != nil {
		// A row that changes something in its own pane: a task ticked off,
		// a task added from the text in the bar. The bar is cleared, since
		// the text was either the new task or a filter that may no longer
		// match; the row number stays, so the next row is under the cursor.
		return Wrap("execute-silent", "swoop-run "+ShellQuote(id)) + "+" + stay()
	}
	if kind == "terminal" {
		// fzf's execute, not execute-silent: the run gets the terminal,
		// keys and screen, until it exits. The kind and title ride along
		// for the usage log, as runCmd sends them: an edit of a note
		// counts as an open of it.
		// What comes back after is Back's, through swoop-nav, since the
		// run may have named a row to land on.
		return Wrap("execute", "env SWOOP_KIND="+ShellQuote(kind)+" SWOOP_TITLE="+ShellQuote(title)+" swoop-run "+ShellQuote(id)) + "+" + Wrap("transform", "swoop-nav back "+ShellQuote(id))
	}
	if kind != "view" {
		return runCmd(id, "")
	}
	st.Stack = append(st.Stack, Frame{Kind: "view", View: id, Title: title, Query: query, Pos: pos})
	return push(title + " > ")
}

// SettingsView is the id of the Settings pane's view, and SettingsTitle
// its prompt.
const (
	SettingsView  = "ext/settings/settings"
	SettingsTitle = "Settings"
)

// Settings decides what the settings key does (cmd+, in the frame, alt+,
// in a terminal): open the Settings pane, as Enter on its row would.
// Inside it already, nothing.
func Settings(st *State, query string, pos int) string {
	if top := st.Top(); top != nil && top.View == SettingsView {
		return "ignore"
	}
	st.Stack = append(st.Stack, Frame{Kind: "view", View: SettingsView, Title: SettingsTitle, Query: query, Pos: pos})
	return push(SettingsTitle + " > ")
}

// enterAction runs the picked action for the actions pane's target. Kind
// "refresh" runs it silently and returns to the pane below, reloaded, so a
// delete shows the list without the entry; anything else ends the launcher
// like Enter on a plain row.
//
// The refresh path runs the bare runner, not runCmd: runCmd is the exit
// command, and it removes the run's files and tells a frame the launcher
// is leaving. Doing that for a delete lost the stack and closed the frame.
//
// Kind "terminal" is refresh with the terminal handed over: execute
// rather than execute-silent, and the preview redrawn after, since the
// run has likely changed what it shows.
func enterAction(st *State, top *Frame, action, kind string, runCmd func(target, action string) string) string {
	target := top.View
	if kind == "terminal" {
		pop := popActions(st, true)
		return Wrap("execute", "swoop-run "+ShellQuote(target)+" "+ShellQuote(action)) + "+" + pop
	}
	if kind != "refresh" {
		return runCmd(target, action)
	}
	pop := popActions(st, false)
	return Wrap("execute-silent", "swoop-run "+ShellQuote(target)+" "+ShellQuote(action)) + "+" + pop
}

// Actions decides what ctrl-k does: push an actions pane for the current
// row. The caller has already checked the row has actions; with no row
// there is nothing to show.
func Actions(st *State, id, kind, title, query string, pos int) string {
	if id == "" {
		return "ignore"
	}
	st.Stack = append(st.Stack, Frame{Kind: "actions", View: id, Title: title, Query: query, Pos: pos})
	// The preview stays on the target while its actions are shown: an
	// action row has nothing of its own to preview.
	return push(title+" actions > ") + "+" + Wrap("change-preview", "swoop-preview "+ShellQuote(id))
}

// AIView is the id of the extension view behind the Ask AI pane, AIPrefix
// what the launcher puts before that extension's row ids, and AITitle
// the pane's prompt.
const (
	AIView   = "ext/ai/ask"
	AIPrefix = "ext/ai/"
	AITitle  = "Ask AI"
)

// PreviewPercent is the preview window's width, in percent of the
// whole. cmd/swoop-nav sets it from the settings file at start; the
// default is what a fresh install shows.
var PreviewPercent = 58

// Window is fzf's preview window for a pane: on the right, the width
// from PreviewPercent, a border on its left. The Ask AI pane's is
// wrapped, because a transcript is prose, and following, so a growing
// answer keeps its end in view. bin/swoop asks swoop-nav for the plain
// one at start, and every pane change sets the one it needs.
func Window(ai bool) string {
	flags := "nowrap"
	if ai {
		flags = "wrap,follow"
	}
	return fmt.Sprintf("right,%d%%,border-left,%s", PreviewPercent, flags)
}

// Divider decides what moving the divider does: the same window the
// pane has now, at the new width. The caller has already moved
// PreviewPercent and saved it.
func Divider(st *State) string {
	top := st.Top()
	return Wrap("change-preview-window", Window(top != nil && top.Kind == "ai"))
}

// Ask decides what Tab does: push the Ask AI pane, keeping whatever is in
// the bar, since it is the prompt about to be sent. Inside the pane Tab
// does nothing. fzf's own matching goes off because the bar is not a
// filter, and the preview window changes shape for the transcript.
func Ask(st *State, query string, pos int) string {
	if top := st.Top(); top != nil && top.Kind == "ai" {
		return "ignore"
	}
	st.Stack = append(st.Stack, Frame{Kind: "ai", View: AIView, Title: AITitle, Query: query, Pos: pos})
	return askPane()
}

// askPane is what opening the Ask AI pane does, from Tab or its keyword.
func askPane() string {
	return strings.Join([]string{
		"disable-search",
		Wrap("change-prompt", AITitle+" > "),
		Wrap("change-preview-window", Window(true)),
		"reload-sync(swoop-nav rows)",
		"first",
	}, "+")
}

// AISendTarget says what Enter in the Ask AI pane should send the bar's
// text to: the row's conversation, or "new" for the New row. The second
// result is false when there is nothing to send, no text or no row. The
// send itself is the caller's, done before the actions below, so that a
// send that fails (the model is still answering the last prompt) can
// leave the bar as it was.
func AISendTarget(st *State, id, query string) (string, bool) {
	top := st.Top()
	if top == nil || top.Kind != "ai" || id == "" || strings.TrimSpace(query) == "" {
		return "", false
	}
	return strings.TrimPrefix(id, AIPrefix), true
}

// LandEvent is the fzf event that puts the cursor where a chain asked,
// once the chain's reload has landed. No chain uses fzf's wait.
//
// Why: pos(N) straight after reload-sync moves the cursor in the list on
// screen, not the one on its way; from a pane of 5 rows, pos(7) lands on
// row 5 of the new list (fzf 0.74, tried with fzf alone). wait holds pos
// back until the list lands, but "while waiting, user input is ignored"
// (fzf's man page): ignored, not queued. A second quick Enter on a
// checklist was lost, and letters typed right after Esc lost the first
// ones.
//
// So a chain that must place the cursor on a new list records a Landing
// and ends in rebind(LandEvent). fzf fires result-final once the reload's
// list is ready; plain "result" also fires before that, for the new text
// against the old list, which is why it is not used. swoop-nav landed
// then places the cursor and unbinds the event again (Landed). Keys
// typed meanwhile are read as usual.
//
// A chain that reloads the pane it is in, a tick, or the return from an
// editor, needs no landing: fzf keeps the cursor's row number across a
// reload, and that is the row the rules ask for.
//
// Until the list lands, keys see the old one. Typing is safe: it only
// edits the bar, and the reload reads the bar. A key that reads the row
// under the cursor is not: a second Enter on a checklist would untick the
// task the first one ticked. So those keys (Enter, a click, ctrl-k, Tab,
// alt-,) start with fzf's wait in bin/swoop, which holds the key until
// the search in flight is done, and then land any pending Landing before
// they read the row (Settle). That is the one wait left, and why: it
// holds only a key that reads a row, only while a list is on its way. A
// key sent while one is held is still dropped, so three Enters inside one
// reload tick two rows; fzf has no way to queue it.
const LandEvent = "result-final"

// Landed is what LandEvent does: the cursor to the Landing's row, the
// preview redrawn if the chain asked, and the event unbound until the
// next chain. query is the bar's text now; typed text since the chain
// means the user has moved on, and the cursor is left alone.
func Landed(st *State, query string) string {
	l := st.Land
	st.Land = nil
	var acts []string
	if l != nil && l.Query == query {
		acts = append(acts, fmt.Sprintf("pos(%d)", max(l.Pos, 1)))
		if l.Refresh {
			acts = append(acts, "refresh-preview")
		}
	}
	return strings.Join(append(acts, "unbind("+LandEvent+")"), "+")
}

// Settle comes first in every key that reads the row under the cursor.
// The key's wait (see LandEvent) held it until the list landed, but the
// LandEvent that places the cursor is queued behind the key, so a
// pending Landing is done here instead, and the key is done again as
// again, its own transform, on the row it lands on. "" when nothing is
// pending.
func Settle(st *State, query, again string) string {
	if st.Land == nil {
		return ""
	}
	return Landed(st, query) + "+" + Wrap("transform", again)
}

// land records where the cursor goes once the reload lands, and returns
// the action that arms LandEvent for it.
func land(st *State, l Landing) string {
	st.Land = &l
	return "rebind(" + LandEvent + ")"
}

// AfterSend is what follows a send: clear the bar, list again so the
// conversation is at the top under New, land on it, and draw it. The
// worker the send started fills the preview from then on.
func AfterSend(st *State) string {
	return strings.Join([]string{
		"clear-query",
		"reload-sync(swoop-nav rows)",
		land(st, Landing{Pos: 2, Refresh: true}),
	}, "+")
}

// push is what entering any pane does: clear the bar first, so the reload
// that follows sees it empty and the pane answers with its "nothing typed
// yet" rows; then fzf's own matching goes off, because inside a pane the
// launcher shows exactly what comes back; then the cursor goes to the top,
// because fzf keeps its row index across a reload, and a menu opened from
// row 2 would otherwise start on its own row 2.
func push(prompt string) string {
	return strings.Join([]string{
		"clear-query",
		"disable-search",
		Wrap("change-prompt", prompt),
		"reload-sync(swoop-nav rows {q})",
		"first",
	}, "+")
}

// stay reloads the pane the user is in, from an empty bar. The cursor
// stays on its row number, which fzf keeps across a reload, so no pos
// and no landing (see LandEvent).
func stay() string {
	return "clear-query+reload-sync(swoop-nav rows {q})"
}

// Back is what follows a terminal row's run. id is the row the run named
// in $SWOOP_LAND, "" for none.
//
// With none, the pane the user is in is reloaded, keeping the bar's
// text; the cursor keeps its row number, as in stay. The text stays
// because nothing was typed into the list: the run had the keys. The
// preview is redrawn, because the file under the cursor has likely
// changed and fzf would otherwise show what it drew before the run. It
// draws the row on screen, which is the row the reload leaves there; a
// different row there after the reload is drawn anyway, as fzf draws
// every row the cursor comes to.
//
// With a row named, the bar is cleared and the cursor lands on that row,
// or on the first if it is not listed. New note needs this: its row is
// made from the bar's text, so with the text kept the cursor came back
// to "New note: <title>", and a second Enter made <title>-2.md.
func Back(st *State, id string) string {
	if id == "" {
		return "reload-sync(swoop-nav rows {q})+refresh-preview"
	}
	return strings.Join([]string{
		"clear-query",
		"reload-sync(swoop-nav rows {q})",
		land(st, Landing{ID: id}),
	}, "+")
}

// Esc decides what Esc does: clear the bar if it has text, otherwise pop
// a pane, otherwise close.
func Esc(st *State, query string) string {
	if query != "" {
		return "clear-query"
	}
	if st.Top() == nil {
		return "abort"
	}
	return popActions(st, false)
}

// popActions removes the top pane and returns the actions that bring the
// pane below it back: its own matching mode and prompt, the text the user
// had typed there, its rows for that text, and the cursor row they were
// on. The text goes back BEFORE the reload: the reload reads the bar, and
// with the pane's old text still in it a view would filter on the wrong
// thing and show nothing. Same rows plus same text give the same order, so
// the row is the one they left. The cursor gets there by a Landing, since
// the rows are another pane's; refresh redraws the preview once there.
func popActions(st *State, refresh bool) string {
	frame := st.Stack[len(st.Stack)-1]
	st.Stack = st.Stack[:len(st.Stack)-1]
	below := st.Top()
	search, prompt, window := "enable-search", RootPrompt, Window(false)
	if below != nil {
		search = "disable-search"
		prompt = below.Title + " > "
		switch below.Kind {
		case "actions":
			prompt = below.Title + " actions > "
		case "ai":
			window = Window(true)
		}
	}
	return strings.Join([]string{
		search,
		Wrap("change-prompt", prompt),
		Wrap("change-preview-window", window),
		Wrap("change-preview", "swoop-preview {1}"),
		Wrap("change-query", frame.Query),
		"reload-sync(swoop-nav rows {q})",
		land(st, Landing{Query: frame.Query, Pos: frame.Pos, Refresh: refresh}),
	}, "+")
}

// Keyed is what a keyword at the start of the root bar names: "def ap"
// names Define, with "ap" left over. cmd/swoop-nav finds the extension
// and asks it for its rows; this package decides what that means.
type Keyed struct {
	// Rest is the bar's text after the keyword and its space.
	Rest string
	// View is the id of the extension's one row when that row is a view,
	// Define Word for Define, or AIView for the Ask AI pane. "" when the
	// extension lists anything else: several rows, or rows that depend on
	// the text, like Calculator's.
	View  string
	Title string // that view row's title, the pane's prompt
}

// Change decides what typing does: ask for new rows, everywhere. Inside a
// pane the extension filters, or the launcher does for an actions pane. At
// the root the apps come from the cache and the extensions are asked with
// the text, which is how a calculator row appears for "2+2" while fzf
// keeps matching the apps itself. In the Ask AI pane the bar is the
// prompt being written, and the list does not change under it.
//
// A keyword at the root, k not nil, scopes the bar to one extension.
// When its one row is a view, the view opens as if Enter had been pressed
// on it, with the rest of the bar as its text: "def ap" is the Define
// pane with "ap" typed. Otherwise the root shows that extension's rows
// alone, and fzf's own matching goes off, because the bar still holds
// the keyword and no row's title does; swoop-nav filters them on the
// rest instead. Every other keystroke at the root turns fzf's matching
// back on, so deleting the space undoes the scope.
//
// The frame a keyword pushes keeps an empty bar, not the keyword: Esc
// comes back to the whole root list, and a keyword given back to the bar
// would only open the pane again.
func Change(st *State, k *Keyed) string {
	top := st.Top()
	if top != nil && top.Kind == "ai" {
		return "ignore"
	}
	if top != nil {
		return "reload-sync(swoop-nav rows {q})"
	}
	if k == nil {
		return "enable-search+reload-sync(swoop-nav rows {q})"
	}
	// change-query fires fzf's change event again; by then the pane is
	// on the stack, and that Change only reloads it.
	switch k.View {
	case "":
		return "disable-search+reload-sync(swoop-nav rows {q})"
	case AIView:
		st.Stack = append(st.Stack, Frame{Kind: "ai", View: AIView, Title: AITitle, Pos: 1})
		return Wrap("change-query", k.Rest) + "+" + askPane()
	}
	st.Stack = append(st.Stack, Frame{Kind: "view", View: k.View, Title: k.Title, Pos: 1})
	return strings.Join([]string{
		Wrap("change-query", k.Rest),
		"disable-search",
		Wrap("change-prompt", k.Title+" > "),
		"reload-sync(swoop-nav rows {q})",
		"first",
	}, "+")
}

// Wrap returns "action<open>arg<close>" with the first delimiter pair that
// does not appear in arg. fzf accepts (), [], {}, <> and ~~ for exactly
// this reason, so a title or a path with a parenthesis in it cannot
// break the action string.
func Wrap(action, arg string) string {
	for _, d := range []string{"()", "[]", "{}", "<>", "~~"} {
		if !strings.ContainsAny(arg, d) {
			return action + d[:1] + arg + d[1:]
		}
	}
	// Every pair appears in the text. Fall back to parentheses with the
	// closing one dropped from the text; a prompt is decoration, not data.
	return action + "(" + strings.ReplaceAll(arg, ")", "") + ")"
}

// ShellQuote single-quotes s for a POSIX shell.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
