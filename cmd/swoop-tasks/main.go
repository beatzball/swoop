// swoop-tasks is the tasks extension: a checklist in one markdown file,
// ~/.local/share/swoop/tasks.md. One root row, Tasks, opens the view:
// open tasks first, the ones due soonest on top, done ones after. Text
// typed in the view becomes an Add task row; date words at its end
// ("tomorrow", "friday", "2026-10-01") become the due date. Enter on a
// task ticks it, and the view stays, reloaded.
//
//	swoop-tasks list                 the root row
//	swoop-tasks view tasks [text]    the view: an Add row for text, the tasks
//	swoop-tasks preview <id>         the task's facts
//	swoop-tasks actions <id>         Done or Undo, Delete, Copy, Edit the list
//	swoop-tasks run <id> [action]    tick, untick, delete, copy, edit, or add
//	swoop-tasks add <text>           a task from the shell, date words and all
//	swoop-tasks parse <text>         the text and its due day, tab-separated
//
// parse is for the reminders extension, which reads the same date words
// and should not read them differently.
//
// A task's id is "t", a unit separator, and its line as it is in the
// file; the Add row's is "add", the separator, and the text. The line
// is the id because it is what stays put: a line number moves when one
// above it is deleted.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
)

// sep joins an id's kind and its text. A control character, so no task
// typed by a person can hold it.
const sep = "\x1f"

const (
	viewID   = "tasks"
	iconOpen = "󰄱"
	iconDone = "󰄲"
	iconAdd  = "󰐕"
)

func main() {
	if len(os.Args) < 2 {
		usageExit()
	}
	arg := func(i int) string {
		if len(os.Args) > i {
			return os.Args[i]
		}
		return ""
	}
	now := time.Now()
	var err error
	switch os.Args[1] {
	case "list":
		err = list(now)
	case "view":
		err = view(arg(3), now)
	case "preview":
		err = preview(arg(2), now)
	case "actions":
		err = actions(arg(2))
	case "run":
		err = run(arg(2), arg(3), now)
	case "add":
		err = addText(strings.Join(os.Args[2:], " "), now)
	case "parse":
		text, due := Split(strings.Join(os.Args[2:], " "), now)
		fmt.Printf("%s\t%s\n", text, due)
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-tasks:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-tasks list | view tasks [text] | preview <id> | actions <id> | run <id> [done|undo|delete|copy|edit] | add <text> | parse <text>")
	os.Exit(2)
}

// list prints the one root row, with a count in its subtitle so the
// launcher says what is waiting without opening the view.
func list(now time.Time) error {
	f, err := load(path())
	if err != nil {
		return err
	}
	return protocol.Write(os.Stdout, []protocol.Item{{ID: viewID, Kind: "view", Icon: iconOpen, Title: "Tasks", Subtitle: summary(f.Tasks(), now)}})
}

// summary is the root row's subtitle: how many are open, and how many of
// those are due today or late.
func summary(ts []Task, now time.Time) string {
	open, due := 0, 0
	for _, t := range ts {
		if t.Done {
			continue
		}
		open++
		if n, ok := daysUntil(t.Due, now); ok && n <= 0 {
			due++
		}
	}
	switch {
	case open == 0:
		return "a checklist; nothing open"
	case due > 0:
		return fmt.Sprintf("%d open, %d due today or late", open, due)
	}
	return fmt.Sprintf("%d open", open)
}

func view(query string, now time.Time) error {
	f, err := load(path())
	if err != nil {
		return err
	}
	return protocol.Write(os.Stdout, rows(f.Tasks(), query, now))
}

// rows is the view. With text typed, the Add row comes first, so Enter
// straight after typing adds, then the tasks that hold every word typed.
// With nothing typed and nothing in the list, a row that says what to do.
//
// Every row is kind "toggle": Enter runs it and the view stays, reloaded,
// so a list is ticked off without leaving it (see internal/nav).
func rows(ts []Task, query string, now time.Time) []protocol.Item {
	var items []protocol.Item
	query = strings.TrimSpace(query)
	// The filter is the text without its date words: "milk friday" is
	// about to add a task, and still finds the milk one already there.
	text, due := Split(query, now)
	if query != "" {
		items = append(items, protocol.Item{ID: "add" + sep + query, Kind: "toggle", Icon: iconAdd, Title: "Add task: " + text, Subtitle: describe(due, now)})
	} else if len(ts) == 0 {
		items = append(items, protocol.Item{ID: "add" + sep, Kind: "toggle", Icon: iconAdd, Title: "Type a task, then Enter", Subtitle: "end it with today, tomorrow, a weekday or a date to give it one"})
	}
	words := strings.Fields(strings.ToLower(text))
	for _, t := range order(ts, now) {
		if !matches(t.Text, words) {
			continue
		}
		it := protocol.Item{ID: "t" + sep + t.Raw, Kind: "toggle", Icon: iconOpen, Title: t.Text}
		if t.Due != "" {
			it.Subtitle = Label(t.Due, now)
		}
		if t.Done {
			// A done task sinks and says so. The protocol has no style
			// field to dim it with; the ticked box and the word do that.
			it.Icon = iconDone
			it.Subtitle = "done"
		}
		items = append(items, it)
	}
	return items
}

// matches says whether text holds every word typed, in any order and
// case: the view filters for itself, since fzf's matching is off in a
// pane.
func matches(text string, words []string) bool {
	lower := strings.ToLower(text)
	for _, w := range words {
		if !strings.Contains(lower, w) {
			return false
		}
	}
	return true
}

// order is the view's order: open tasks with a due day, soonest first,
// so overdue and today are on top; then open tasks without one; then
// done ones. Within each, file order, which is the order they were added.
func order(ts []Task, now time.Time) []Task {
	out := append([]Task(nil), ts...)
	rank := func(t Task) int {
		switch {
		case t.Done:
			return 2
		case t.Due == "":
			return 1
		}
		return 0
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if rank(a) != rank(b) {
			return rank(a) < rank(b)
		}
		if rank(a) == 0 && a.Due != b.Due {
			return a.Due < b.Due
		}
		return false
	})
	return out
}

// resolve reads an id back: the Add row's text, or the task it names.
func resolve(f *File, id string) (text string, t Task, isAdd bool, err error) {
	kind, rest, _ := strings.Cut(id, sep)
	switch kind {
	case "add":
		return rest, Task{}, true, nil
	case "t":
		if t, ok := f.find(rest); ok {
			return "", t, false, nil
		}
		return "", Task{}, false, errors.New("that task is no longer in the file")
	case viewID:
		return "", Task{}, false, nil
	}
	return "", Task{}, false, fmt.Errorf("no task %q", id)
}

func preview(id string, now time.Time) error {
	if id == viewID {
		fmt.Println("A checklist, kept in", tilde(path()))
		fmt.Println()
		fmt.Println("Enter, then type a task. End it with today, tomorrow, a")
		fmt.Println("weekday or a date like 2026-10-01 and that is its due day.")
		fmt.Println("Enter on a task ticks it; ctrl-k undoes, deletes, copies.")
		return nil
	}
	f, err := load(path())
	if err != nil {
		return err
	}
	text, t, isAdd, err := resolve(f, id)
	if err != nil {
		return err
	}
	if isAdd {
		task, due := Split(text, now)
		if task == "" {
			fmt.Println("Type a task, then Enter.")
			fmt.Println()
			fmt.Println("buy milk tomorrow     due tomorrow")
			fmt.Println("send the report fri  due Friday")
			fmt.Println("renew 2026-10-01      due that day")
			return nil
		}
		fmt.Println("Enter adds:")
		fmt.Println()
		fmt.Println(task)
		fmt.Println()
		fmt.Println(describe(due, now))
		return nil
	}
	fmt.Println(t.Text)
	fmt.Println()
	if t.Due != "" {
		fmt.Println("Due", Label(t.Due, now)+",", t.Due)
	} else {
		fmt.Println("No due date")
	}
	if t.Done {
		fmt.Println("Done. Enter or ctrl-k Undo opens it again.")
	} else {
		fmt.Println("Open. Enter ticks it.")
	}
	fmt.Println()
	fmt.Println("Tasks live in", tilde(path()))
	return nil
}

// tilde shows the home directory as ~, so no home path reaches the
// screen.
func tilde(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// actions is the ctrl-k menu of a task. Done or Undo and Delete are
// refreshes: they change the file and go back to the view, reloaded.
// Copy ends the launcher, like a copy anywhere else. Edit the list opens
// the whole file in the editor setting, inside the panel, and comes back
// to the view (kind "terminal"); it is also the Tasks row's one action,
// at the root. The Add row and the hint have no menu: there is nothing
// yet to act on.
func actions(id string) error {
	edit := protocol.Item{ID: "edit", Kind: "terminal", Icon: iconOpen, Title: "Edit the list", Subtitle: "the whole file, in " + strings.Join(settings.Editor(), " ")}
	if id == viewID {
		return protocol.Write(os.Stdout, []protocol.Item{edit})
	}
	f, err := load(path())
	if err != nil {
		return err
	}
	_, t, isAdd, err := resolve(f, id)
	if err != nil || isAdd {
		return err
	}
	tick := protocol.Item{ID: "done", Kind: "refresh", Icon: iconDone, Title: "Mark as Done", Subtitle: "Enter"}
	if t.Done {
		tick = protocol.Item{ID: "undo", Kind: "refresh", Icon: iconOpen, Title: "Undo", Subtitle: "Open the task again"}
	}
	return protocol.Write(os.Stdout, []protocol.Item{
		tick,
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: "Remove the task from the file"},
		{ID: "copy", Kind: "action", Icon: "", Title: "Copy", Subtitle: "The task's text"},
		edit,
	})
}

// run does what Enter or an action asked. Enter on a task flips it, so
// Enter on a done one opens it again: the view is a checklist, and a
// box that only ticks one way would be a trap.
func run(id, action string, now time.Time) error {
	if action == "edit" {
		// The file as it is, whichever row asked: the list is one file.
		// Its folder first, or an editor on a list not yet written could
		// not save it.
		if err := os.MkdirAll(filepath.Dir(path()), 0o700); err != nil {
			return err
		}
		return settings.Edit(path())
	}
	f, err := load(path())
	if err != nil {
		return err
	}
	text, t, isAdd, err := resolve(f, id)
	if err != nil {
		return err
	}
	if isAdd {
		return addTo(f, text, now)
	}
	if id == viewID {
		return nil
	}
	switch action {
	case "":
		f.SetDone(t, !t.Done)
	case "done":
		f.SetDone(t, true)
	case "undo":
		f.SetDone(t, false)
	case "delete":
		f.Delete(t)
	case "copy":
		if err := paste.Copy(t.Text); err != nil {
			return err
		}
		paste.Tell("Copied " + paste.Short(t.Text))
		return nil
	default:
		return fmt.Errorf("no action %q for a task", action)
	}
	return f.Save()
}

// addText is `swoop-tasks add`, from the shell.
func addText(text string, now time.Time) error {
	f, err := load(path())
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		usageExit()
	}
	return addTo(f, text, now)
}

// addTo adds text as a task, its date words read. Empty text is the
// hint row's Enter, and does nothing.
func addTo(f *File, text string, now time.Time) error {
	task, due := Split(text, now)
	if task == "" {
		return nil
	}
	f.Add(task, due)
	return f.Save()
}
