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
//   - A row of kind "group" is a header over the rows below it, Today
//     over the tasks due today. It has nothing to run, and the cursor
//     never rests on it: Up and Down step over it, a list opens or
//     reloads with the cursor on the row after it, and a click on it
//     puts the cursor back where it was. See Step.
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
//   - A key an extension claims opens that extension's view from
//     anywhere, as Enter on its row would. Inside that view the key does
//     nothing. See Key.
//   - A view may say its bar is a prompt, not a filter: typing changes
//     nothing in the list, and Enter sends the text to the row under the
//     cursor, a question to a conversation. A key that opens such a view
//     keeps the bar's text, since it is the prompt about to be sent.
//   - A view may say what preview window it wants: wrapped for prose, a
//     width of its own. It is put back when the pane is left. See Window.
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

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/protocol"
)

// Frame is one pushed pane.
type Frame struct {
	// Kind is "view" for an extension's pane, "actions" for a row's
	// actions.
	Kind string `json:"kind"`
	// View is the id of the row that opened it: the view row for a view,
	// the target row for an actions pane.
	View  string `json:"view"`
	Title string `json:"title"` // its title, used as the prompt
	Query string `json:"query"` // the bar's text at the moment of Enter
	Pos   int    `json:"pos"`   // the cursor row at the moment of Enter, 1-based
	// Pane is what the view says about its own pane, from the extension's
	// views file: a bar that is a prompt, a preview window of its own.
	// Zero for a plain list, and for an actions pane.
	Pane ext.Pane `json:"pane"`
}

// State is the stack. Empty means the root list.
type State struct {
	Stack []Frame `json:"stack"`
	// Land is where the cursor goes once the reload in flight lands, or
	// nil. See LandEvent.
	Land *Landing `json:"land,omitempty"`
	// Rest is the row the cursor last came to rest on, 1-based, or 0 when
	// not known. A click on a header goes back to it. See Step.
	Rest int `json:"rest,omitempty"`
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
// cmd/swoop-nav). "" action means the default. pane is what a view row's
// view says about its pane; the caller reads it for a view row only.
func Enter(st *State, id, kind, title, query string, pos int, pane ext.Pane, runCmd func(target, action string) string) string {
	if id == "" {
		return "ignore"
	}
	if top := st.Top(); top != nil && top.Kind == "actions" {
		return enterAction(st, top, id, kind, runCmd)
	}
	if top := st.Top(); top != nil && top.Pane.Prompt {
		// The bar is a prompt. The caller sends first; see Sends. Enter
		// here never runs a row or pushes a pane.
		return "ignore"
	}
	if kind == "group" {
		// A header over the rows below it. Nothing to run, at the root
		// or in a view. The cursor is on one only when the key came
		// before it was moved off, and the caller has tried that once
		// already (OffHeader).
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
	return open(st, Frame{Kind: "view", View: id, Title: title, Query: query, Pos: pos, Pane: pane}, false)
}

// Key decides what a key an extension claims does: open the view the
// claim names, as Enter on its row would. view is that view's id, title
// its prompt, pane what the view says about its pane. Inside the view
// already, nothing. A view whose bar is a prompt keeps the bar's text,
// since it is the prompt about to be sent; any other opens with the bar
// empty, and gives the text back on the way out.
func Key(st *State, view, title string, pane ext.Pane, query string, pos int) string {
	if top := st.Top(); top != nil && top.View == view {
		return "ignore"
	}
	return open(st, Frame{Kind: "view", View: view, Title: title, Query: query, Pos: pos, Pane: pane}, pane.Prompt)
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

// PreviewPercent is the preview window's width, in percent of the
// whole. cmd/swoop-nav sets it from the settings file at start; the
// default is what a fresh install shows.
var PreviewPercent = 58

// Window is fzf's preview window for a pane: on the right, a border on
// its left, the width from PreviewPercent, lines cut at the edge. A view
// may ask for its own in its views file: wrapped, because a transcript
// is prose; following, so a growing answer keeps its end in view; a
// width of its own. bin/swoop asks swoop-nav for the plain one at start,
// and every pane change sets the one it needs.
func Window(p ext.Pane) string {
	percent, flags := PreviewPercent, "nowrap"
	if p.Percent > 0 {
		percent = p.Percent
	}
	if p.Wrap {
		flags = "wrap"
	}
	if p.Follow {
		flags += ",follow"
	}
	return fmt.Sprintf("right,%d%%,border-left,%s", percent, flags)
}

// shape is the pane whose preview window is on the screen: the view the
// user is in, or the one under an actions pane, which keeps the window
// of the pane it was opened from. The plain one at the root.
func shape(st *State) ext.Pane {
	for i := len(st.Stack) - 1; i >= 0; i-- {
		if st.Stack[i].Kind != "actions" {
			return st.Stack[i].Pane
		}
	}
	return ext.Pane{}
}

// OwnWidth says whether the pane on the screen has a preview width of
// its own. The divider keys move the user's width, which such a pane
// does not use, so there they do nothing.
func OwnWidth(st *State) bool {
	return shape(st).Percent > 0
}

// Divider decides what moving the divider does: the same window the
// pane has now, at the new width. The caller has already moved
// PreviewPercent and saved it.
func Divider(st *State) string {
	return Wrap("change-preview-window", Window(shape(st)))
}

// open pushes the view's frame and returns what entering its pane does.
// The bar is cleared first, unless keep says its text stays, so the
// reload that follows sees it empty and the pane answers with its
// "nothing typed yet" rows; then fzf's own matching goes off, because
// inside a pane the launcher shows exactly what comes back; the preview
// window changes when the view asks for another than the one on screen;
// then the cursor goes to the top, because fzf keeps its row index
// across a reload. Last, LandEvent is armed: the first row may be a
// header, Past due over the tasks, and the cursor is stepped off it once
// the rows are there (Landed).
//
// A pane whose bar is a prompt is asked for its rows without the text,
// which is not a filter there, and does not arm LandEvent: its own
// worker reloads it many times a second while an answer arrives.
func open(st *State, f Frame, keep bool) string {
	before := Window(shape(st))
	st.Stack = append(st.Stack, f)
	var acts []string
	if !keep {
		acts = append(acts, "clear-query")
	}
	acts = append(acts, "disable-search", Wrap("change-prompt", f.Title+" > "))
	if window := Window(f.Pane); window != before {
		acts = append(acts, Wrap("change-preview-window", window))
	}
	if f.Pane.Prompt {
		acts = append(acts, "reload-sync(swoop-nav rows)", "first")
	} else {
		acts = append(acts, "reload-sync(swoop-nav rows {q})", "first", "rebind("+LandEvent+")")
	}
	return strings.Join(acts, "+")
}

// Sends says whether Enter in the pane the user is in sends the bar's
// text to the row id: the bar is a prompt, there is a row, and there is
// text. The send itself is the caller's, done before the actions of
// AfterSend, so that a send that fails (the model is still answering the
// last prompt) can leave the bar as it was.
func Sends(st *State, id, query string) bool {
	top := st.Top()
	return top != nil && top.Kind == "view" && top.Pane.Prompt && id != "" && strings.TrimSpace(query) != ""
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
// Inside a view the event stays bound once its chain is done, and every
// list that lands there is looked at: a header under the cursor is
// stepped off (Step). A tick keeps the row number, and that number can
// be the next group's header now; a view's own reload, asked of fzf from
// outside when its rows arrive late, puts a header on the first row. At
// the root and in the other panes it is unbound again, so it costs
// nothing on a keystroke there.
//
// Until the list lands, keys see the old one. Typing is safe: it only
// edits the bar, and the reload reads the bar. A key that reads the row
// under the cursor is not: a second Enter on a checklist would untick the
// task the first one ticked. So those keys (Enter, a click, ctrl-k, a key
// an extension claims) start with fzf's wait in bin/swoop, which holds the key until
// the search in flight is done, and then land any pending Landing before
// they read the row (Settle). That is the one wait left, and why: it
// holds only a key that reads a row, only while a list is on its way. A
// key sent while one is held is still dropped, so three Enters inside one
// reload tick two rows; fzf has no way to queue it.
const LandEvent = "result-final"

// Landed is what LandEvent does: the cursor to the Landing's row, off a
// header if that is one, the preview redrawn if the chain asked, and the
// event unbound until the next chain, except inside a view (see
// LandEvent). query is the bar's text now; typed text since the chain
// means the user has moved on, and the cursor is left where the typing
// put it. kind and pos are the row under the cursor now, before any of
// this: with no Landing to go to, a row that is not a header is where
// the cursor rests, and nothing is left to do. "" is nothing to do.
func Landed(st *State, query, kind string, pos int) string {
	l := st.Land
	st.Land = nil
	var acts []string
	placed := l != nil && l.Query == query
	switch {
	case placed:
		// What row that is, is known only once the cursor is there.
		acts = append(acts, fmt.Sprintf("pos(%d)", max(l.Pos, 1)), stepAgain("down", 0, false))
		if l.Refresh {
			acts = append(acts, "refresh-preview")
		}
	case kind == "group":
		acts = append(acts, stepAgain("down", 0, false))
	case pos > 0:
		st.Rest = pos
	}
	if top := st.Top(); top == nil || top.Kind != "view" || top.Pane.Prompt {
		acts = append(acts, "unbind("+LandEvent+")")
	}
	return strings.Join(acts, "+")
}

// Step keeps the cursor off a header. It runs after every move of the
// cursor: bin/swoop binds each key that moves it to the move and then
// swoop-nav step, and Landed and OffHeader end in it. dir is the way the
// cursor was going, "up" or "down"; kind and pos are the row it is on
// now; from is the row it was on before the last move Step itself asked
// for, 0 on the first call.
//
// On a row that is not a header there is nothing to do: that row is
// where the cursor rests, and it is remembered for a click (Click). On a
// header the cursor goes one more the same way, and Step is asked again.
// A header the move cannot leave, the first row going up, or the last
// going down, turns the cursor round once, so the ends of the list do
// not trap it; turned says that has happened, and a second end stops
// there, which takes a list of headers only, and the contract rules that
// out.
//
// fzf has no row the cursor cannot be on, and no event that says which
// way it moved, so this is a process on every Up and Down. A key held
// down repeats some 30 times a second, and swoop-nav answers in a few
// milliseconds.
func Step(st *State, dir, kind string, pos, from int, turned bool) string {
	if kind != "group" {
		// With a Landing in flight the cursor is not at rest yet, and
		// swoop-nav rows may be saving the state (Find): leave it be.
		if st.Land == nil && pos > 0 {
			st.Rest = pos
		}
		return ""
	}
	if pos == from || (dir == "up" && pos <= 1) {
		if turned {
			return ""
		}
		turned = true
		if dir == "up" {
			dir = "down"
		} else {
			dir = "up"
		}
	}
	return dir + "+" + stepAgain(dir, pos, turned)
}

// stepAgain is the action that asks Step about the row the cursor is on
// once the actions before it are done. fzf fills {2} in when it runs the
// transform, not when it reads the chain, so it is that row's kind.
func stepAgain(dir string, from int, turned bool) string {
	cmd := fmt.Sprintf("swoop-nav step %s {2} %d", dir, from)
	if turned {
		cmd += " turned"
	}
	return Wrap("transform", cmd)
}

// OffHeader is what a key that reads the row under the cursor does when
// that row is a header: the cursor goes to the row after it, and the key
// is done again there, as again, its own transform. The cursor does not
// rest on a header, so this is a key that came before LandEvent had
// moved it off one: the second of two quick Enters, when the first
// ticked the last task of a group and the next group's header took its
// row number. again must say it is the second try, so a header that
// cannot be left ends in Enter's "ignore" and not in a loop.
func OffHeader(again string) string {
	return stepAgain("down", 0, false) + "+" + Wrap("transform", again)
}

// Click is what a click on a header does: nothing, and the cursor back
// where it was. fzf has moved the cursor to the clicked row before the
// binding runs and does not say from where, which is why Step remembers
// the row. Where it is not known, the row after the header.
func Click(st *State) string {
	if st.Rest < 1 {
		return stepAgain("down", 0, false)
	}
	// fzf's own matching at the root moves the cursor without a word to
	// Step, so the row remembered may be a header by now.
	return fmt.Sprintf("pos(%d)+", st.Rest) + stepAgain("down", 0, false)
}

// Headers gives each row of kind "group" the look of a header: its title
// at the left edge, in the icon column, dimmed, so the icons of the rows
// under it line up below its first letter and not to its left. The
// subtitle, its count, stays where it is. fzf shows "icon title", so the
// title goes out in the icon's place and the title's own is left empty;
// fzf matches on both at the root, so a header is still found by its
// name. It is done here, as the rows go to fzf, so every extension that
// prints a group row gets it.
func Headers(items []protocol.Item) {
	for i, it := range items {
		if it.Kind == "group" && it.Title != "" {
			items[i].Icon = "\x1b[2m" + it.Title + "\x1b[22m"
			items[i].Title = ""
		}
	}
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
	acts := Wrap("transform", again)
	if landed := Landed(st, query, "", 0); landed != "" {
		acts = landed + "+" + acts
	}
	return acts
}

// land records where the cursor goes once the reload lands, and returns
// the action that arms LandEvent for it.
func land(st *State, l Landing) string {
	st.Land = &l
	return "rebind(" + LandEvent + ")"
}

// AfterSend is what follows a send: clear the bar, list again, since the
// send may have made a row or moved one, land on the row the send named
// in $SWOOP_LAND, id, and draw it. With no row named, "", the cursor
// stays on its row number, pos. The worker the send started fills the
// preview from then on.
func AfterSend(st *State, id string, pos int) string {
	l := Landing{ID: id, Refresh: true}
	if id == "" {
		l.Pos = pos
	}
	return strings.Join([]string{
		"clear-query",
		"reload-sync(swoop-nav rows)",
		land(st, l),
	}, "+")
}

// push is what entering an actions pane does, the same steps as open: the
// bar cleared, fzf's own matching off, the cursor to the top, since a
// menu opened from row 2 would otherwise start on its own row 2, and
// LandEvent armed.
func push(prompt string) string {
	return strings.Join([]string{
		"clear-query",
		"disable-search",
		Wrap("change-prompt", prompt),
		"reload-sync(swoop-nav rows {q})",
		"first",
		"rebind(" + LandEvent + ")",
	}, "+")
}

// stay reloads the pane the user is in, from an empty bar. The cursor
// stays on its row number, which fzf keeps across a reload, so no pos
// and no landing (see LandEvent). A header on that row number is stepped
// off by the event a view keeps bound.
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
	// The window of the pane that is back: a view's own, if it has one.
	search, prompt, window := "enable-search", RootPrompt, Window(shape(st))
	if below != nil {
		search = "disable-search"
		prompt = below.Title + " > "
		if below.Kind == "actions" {
			prompt = below.Title + " actions > "
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
	// View is the id of the view the keyword opens: the one a key of the
	// extension's opens, or its one row when that row is a view, Define
	// Word for Define. "" when the extension claims no key and lists
	// anything else: several rows, or rows that depend on the text, like
	// Calculator's.
	View  string
	Title string   // that view's title, the pane's prompt
	Pane  ext.Pane // what that view says about its pane
}

// Change decides what typing does: ask for new rows, everywhere. Inside a
// pane the extension filters, or the launcher does for an actions pane. At
// the root the rows kept for the run come from their file and the other
// extensions are asked with the text, which is how a calculator row appears
// for "2+2" while fzf keeps matching the kept rows itself. Inside a pane the
// cursor goes to the first row, as it does at the root, where fzf's own
// matching moves it. In a pane whose bar is a prompt the text is being
// written to be sent, and the list does not change under it.
//
// A keyword at the root, k not nil, scopes the bar to one extension.
// When it names a view, the view opens as if Enter had been pressed on
// its row, with the rest of the bar as its text: "def ap" is the Define
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
	if top != nil && top.Pane.Prompt {
		return "ignore"
	}
	if top != nil {
		// The cursor goes to the top with the new rows: a view puts what
		// the text asks for first, Add task for a task typed. fzf keeps
		// the row number across a reload, so from a row further down,
		// Enter after typing landed on whatever that number now held.
		// A header on the first row is stepped off once the rows land,
		// by the event a view keeps bound (see LandEvent).
		return "reload-sync(swoop-nav rows {q})+first"
	}
	if k == nil {
		return "enable-search+reload-sync(swoop-nav rows {q})"
	}
	// change-query fires fzf's change event again; by then the pane is
	// on the stack, and that Change only reloads it.
	if k.View == "" {
		return "disable-search+reload-sync(swoop-nav rows {q})"
	}
	return Wrap("change-query", k.Rest) + "+" + open(st, Frame{Kind: "view", View: k.View, Title: k.Title, Pos: 1, Pane: k.Pane}, true)
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
