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
//   - Tab opens the Ask AI pane from anywhere, and keeps the bar's text.
//     The bar there is a prompt, not a filter: Enter sends it to the
//     conversation under the cursor, or to a new one, and the answer
//     arrives on the right. Tab inside the pane does nothing.
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
	if kind != "view" {
		return runCmd(id, "")
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
		return runCmd(target, action)
	}
	pop := popActions(st)
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

// PreviewWindow is fzf's preview window as bin/swoop sets it, and
// AIPreviewWindow the same window while the Ask AI pane is up: wrapped,
// because a transcript is prose, and following, so a growing answer
// keeps its end in view.
const (
	PreviewWindow   = "right,45%,border-left,nowrap"
	AIPreviewWindow = "right,45%,border-left,wrap,follow"
)

// Ask decides what Tab does: push the Ask AI pane, keeping whatever is in
// the bar, since it is the prompt about to be sent. Inside the pane Tab
// does nothing. fzf's own matching goes off because the bar is not a
// filter, and the preview window changes shape for the transcript.
func Ask(st *State, query string, pos int) string {
	if top := st.Top(); top != nil && top.Kind == "ai" {
		return "ignore"
	}
	st.Stack = append(st.Stack, Frame{Kind: "ai", View: AIView, Title: AITitle, Query: query, Pos: pos})
	return strings.Join([]string{
		"disable-search",
		Wrap("change-prompt", AITitle+" > "),
		Wrap("change-preview-window", AIPreviewWindow),
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

// AfterSend is what follows a send: clear the bar, list again so the
// conversation is at the top under New, land on it, and draw it. The
// worker the send started fills the preview from then on.
func AfterSend() string {
	return strings.Join([]string{
		"clear-query",
		"reload-sync(swoop-nav rows)",
		"wait",
		"pos(2)",
		"refresh-preview",
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
	search, prompt, window := "enable-search", RootPrompt, PreviewWindow
	if below != nil {
		search = "disable-search"
		prompt = below.Title + " > "
		switch below.Kind {
		case "actions":
			prompt = below.Title + " actions > "
		case "ai":
			window = AIPreviewWindow
		}
	}
	return strings.Join([]string{
		search,
		Wrap("change-prompt", prompt),
		Wrap("change-preview-window", window),
		Wrap("change-preview", "swoop-preview {1}"),
		Wrap("change-query", frame.Query),
		"reload-sync(swoop-nav rows {q})",
		"wait",
		fmt.Sprintf("pos(%d)", frame.Pos),
	}, "+")
}

// Change decides what typing does: ask for new rows, everywhere. Inside a
// pane the extension filters, or the launcher does for an actions pane. At
// the root the apps come from the cache and the extensions are asked with
// the text, which is how a calculator row appears for "2+2" while fzf
// keeps matching the apps itself. In the Ask AI pane the bar is the
// prompt being written, and the list does not change under it.
func Change(st *State) string {
	if top := st.Top(); top != nil && top.Kind == "ai" {
		return "ignore"
	}
	return "reload-sync(swoop-nav rows {q})"
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
