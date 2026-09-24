// Package nav is the launcher's memory of where the user is: a stack of
// panes. fzf itself only ever shows one list; every "go into Define Word"
// and every "Esc back out" is fzf swapping that list on the instructions
// this package prints. The rules, from the navigation design issue:
//
//   - Esc with text in the bar clears the text. Esc with an empty bar
//     closes at the root, or pops one pane inside a view.
//   - Enter on a row of kind "view" pushes a pane: the view's rows, its
//     own prompt, an empty bar, and fzf's own matching turned off so the
//     extension does the filtering. Enter on any other row runs it.
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
	View  string `json:"view"`  // the id of the row that opened it, "ext/<name>/<id>"
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
// row at that moment. A view row pushes a pane and the returned actions
// load it; any other row becomes the run command; no row at all is
// ignored. runCmd is the shell command that performs the run, already
// quoted.
func Enter(st *State, id, kind, title, query string, pos int, runCmd string) string {
	if id == "" {
		return "ignore"
	}
	if kind != "view" {
		// become replaces fzf with the runner, so nothing is left behind.
		// The colon form takes the rest of the string, which keeps any
		// character in the command safe.
		return "become:" + runCmd
	}
	st.Stack = append(st.Stack, Frame{View: id, Title: title, Query: query, Pos: pos})
	// clear-query first, so the reload that follows sees an empty bar and
	// the view answers with its "nothing typed yet" rows. Then fzf's own
	// matching goes off: inside a view the extension filters, and the
	// launcher shows exactly what it returns.
	return strings.Join([]string{
		"clear-query",
		"disable-search",
		wrap("change-prompt", title+" > "),
		"reload-sync(swoop-nav rows {q})",
	}, "+")
}

// Esc decides what Esc does: clear the bar if it has text, otherwise pop
// a pane, otherwise close.
func Esc(st *State, query string) string {
	if query != "" {
		return "clear-query"
	}
	top := st.Top()
	if top == nil {
		return "abort"
	}
	frame := *top
	st.Stack = st.Stack[:len(st.Stack)-1]
	// Matching is back on before the root rows are reloaded; then the old
	// text goes back into the bar, fzf is told to finish that search
	// (wait), and the cursor lands on the saved row. Same rows plus same
	// text give the same order, so the row is the one the user left.
	return strings.Join([]string{
		"enable-search",
		wrap("change-prompt", RootPrompt),
		"reload-sync(swoop-nav rows {q})",
		wrap("change-query", frame.Query),
		"wait",
		fmt.Sprintf("pos(%d)", frame.Pos),
	}, "+")
}

// Change decides what typing does: ask for new rows, everywhere. Inside a
// view the extension filters. At the root the apps come from the cache and
// the extensions are asked with the text, which is how a calculator row
// appears for "2+2" while fzf keeps matching the apps itself.
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
