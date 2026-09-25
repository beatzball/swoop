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
//   - Popping restores the text and the cursor row the user left.
//
// The functions here return fzf action strings. They do no I/O of their
// own except through Load and Save, so they can be tested without fzf.
package nav

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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
}

// State is the stack. Empty means the root list.
type State struct {
	Stack []Frame `json:"stack"`
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

// Save writes the state file.
func Save(path string, st *State) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// RootPrompt is the bar's prompt at the root.
const RootPrompt = "  "

// Enter decides what Enter does for the current row. id, kind, and title
// are the row's fields; query and pos are the bar's text and the cursor
// row at that moment. A view row pushes a pane; an action row runs its
// action; any other row becomes the run command; no row at all is
// ignored. runCmd turns a target id and an action into the shell command
// that performs it, already quoted, "" action meaning the default.
func Enter(st *State, id, kind, title, query string, pos int, runCmd func(target, action string) string) string {
	if id == "" {
		return "ignore"
	}
	if top := st.Top(); top != nil && top.Kind == "actions" {
		return enterAction(st, top, id, kind, runCmd)
	}
	if kind != "view" {
		// become replaces fzf with the runner, so nothing is left behind.
		// The colon form takes the rest of the string, which keeps any
		// character in the command safe.
		return "become:" + runCmd(id, "")
	}
	st.Stack = append(st.Stack, Frame{Kind: "view", View: id, Title: title, Query: query, Pos: pos})
	return push(title + " > ")
}

// enterAction runs the picked action for the actions pane's target. Kind
// "refresh" runs it silently and returns to the pane below, reloaded, so a
// delete shows the list without the entry; anything else ends the launcher
// like Enter on a plain row.
//
// The refresh path runs the bare runner, not runCmd: runCmd is the exit
// command, and it removes the run's files and tells a frame the launcher
// is leaving. Doing that for a delete lost the stack and closed the frame.
func enterAction(st *State, top *Frame, action, kind string, runCmd func(target, action string) string) string {
	target := top.View
	if kind != "refresh" {
		return "become:" + runCmd(target, action)
	}
	pop := popActions(st)
	return "execute-silent(swoop-run " + ShellQuote(target) + " " + ShellQuote(action) + ")+" + pop
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
	return push(title+" actions > ") + "+" + wrap("change-preview", "swoop-preview "+ShellQuote(id))
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
		wrap("change-prompt", prompt),
		"reload-sync(swoop-nav rows {q})",
		"first",
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
	return popActions(st)
}

// popActions removes the top pane and returns the actions that bring the
// pane below it back: its own matching mode and prompt, the text the user
// had typed there, its rows for that text, and the cursor row they were
// on. The text goes back BEFORE the reload: the reload reads the bar, and
// with the pane's old text still in it a view would filter on the wrong
// thing and show nothing. Same rows plus same text give the same order, so
// the row is the one they left.
func popActions(st *State) string {
	frame := st.Stack[len(st.Stack)-1]
	st.Stack = st.Stack[:len(st.Stack)-1]
	below := st.Top()
	search, prompt := "enable-search", RootPrompt
	if below != nil {
		search = "disable-search"
		prompt = below.Title + " > "
		if below.Kind == "actions" {
			prompt = below.Title + " actions > "
		}
	}
	return strings.Join([]string{
		search,
		wrap("change-prompt", prompt),
		wrap("change-preview", "swoop-preview {1}"),
		wrap("change-query", frame.Query),
		"reload-sync(swoop-nav rows {q})",
		"wait",
		fmt.Sprintf("pos(%d)", frame.Pos),
	}, "+")
}

// Change decides what typing does: ask for new rows, everywhere. Inside a
// pane the extension filters, or the launcher does for an actions pane. At
// the root the apps come from the cache and the extensions are asked with
// the text, which is how a calculator row appears for "2+2" while fzf
// keeps matching the apps itself.
func Change(*State) string {
	return "reload-sync(swoop-nav rows {q})"
}

// wrap returns "action<open>arg<close>" with the first delimiter pair that
// does not appear in arg. fzf accepts (), [], {}, <> and ~~ for exactly
// this reason, so a title with a parenthesis in it cannot break the
// action string.
func wrap(action, arg string) string {
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
