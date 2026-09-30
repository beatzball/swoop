// swoop-tasks is the tasks extension: a checklist in one markdown file,
// ~/.local/share/swoop/tasks.md. One root row, Tasks, opens the view:
// the open tasks under headers by when they are due, Past due to
// Unscheduled, and a Done row last that opens the done ones as a view of
// their own. Text typed in the view becomes an Add task row; date words
// at its end ("tomorrow", "friday", "2026-10-01") become the due date.
// Enter on a task ticks it, and the view stays, reloaded.
//
//	swoop-tasks list                 the root row
//	swoop-tasks view tasks [text]    the view: an Add row for text, the open tasks
//	swoop-tasks view done [text]     the done tasks, under a Back to open row
//	swoop-tasks preview <id>         the task's facts
//	swoop-tasks actions <id>         Done or Undo, Delete, Copy, Edit the list
//	swoop-tasks run <id> [action]    tick, untick, delete, copy, edit, or add
//	swoop-tasks add <text>           a task from the shell, date words and all
//	swoop-tasks parse <text>         the text and its due day, tab-separated
//	swoop-tasks group                stdin's lines, a due day first in each, in
//	                                 the view's order, the group's name in front
//
// parse and group are for the reminders extension, which reads the same
// date words and draws the same headers, and should not do either
// differently.
//
// A task's id is "t", a unit separator, and its line as it is in the
// file; the Add row's is "add", the separator, and the text. The line
// is the id because it is what stays put: a line number moves when one
// above it is deleted. A header's id is "group", the separator, and its
// name; the Done row's is "done" and Back to open's is "open". None of
// the three names a task, so none has actions, and running one does
// nothing.
//
// SWOOP_TASKS_NOW, as 2026-09-14T10:30, is the clock for a test that
// drives the real tool: which group a task is under depends on the day.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/match"
	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
)

// sep joins an id's kind and its text. A control character, so no task
// typed by a person can hold it.
const sep = "\x1f"

const (
	viewID   = "tasks"
	doneID   = "done"  // the Done row, and the view it opens
	openID   = "open"  // Back to open, in that view
	groupID  = "group" // a header's id starts with it
	iconOpen = "󰄱"
	iconDone = "󰄲"
	iconAdd  = "󰐕"
	iconBack = "󰁍"
)

// envNow names the clock a test pins; see the top of the file.
const envNow = "SWOOP_TASKS_NOW"

// clock is now, or the minute in SWOOP_TASKS_NOW when that is set and
// reads as one.
func clock() time.Time {
	if t, err := time.ParseInLocation("2006-01-02T15:04", os.Getenv(envNow), time.Local); err == nil {
		return t
	}
	return time.Now()
}

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
	now := clock()
	var err error
	switch os.Args[1] {
	case "list":
		err = list(now)
	case "view":
		err = view(arg(2), arg(3), now)
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
	case "group":
		err = groupLines(os.Stdin, os.Stdout, now, settings.WeekStart())
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-tasks:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-tasks list | view tasks|done [text] | preview <id> | actions <id> | run <id> [done|undo|delete|copy|edit] | add <text> | parse <text> | group")
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

func view(id, query string, now time.Time) error {
	f, err := load(path())
	if err != nil {
		return err
	}
	if id == doneID {
		return protocol.Write(os.Stdout, doneRows(f.Tasks(), query, loadDone(donePath())))
	}
	return protocol.Write(os.Stdout, rows(f.Tasks(), query, now, settings.WeekStart()))
}

// rows is the view: the open tasks under a header for each group that
// has any, in the groups' order, and the Done row last. With text typed,
// the Add row comes first, so Enter straight after typing adds, then the
// open tasks that match every word typed (internal/match); a group with
// none of them has no header. The groups and the days keep their order;
// among tasks due the same day the best match comes first. With nothing typed and nothing open, a row that says what to do.
//
// A task row is kind "toggle": Enter runs it and the view stays, reloaded,
// so a list is ticked off without leaving it (see internal/nav). A header
// is kind "group", which Enter ignores. The Done row is kind "view": the
// done tasks are a pane of their own on top of this one, so Esc comes
// back here too, to the same text and row.
func rows(ts []Task, query string, now time.Time, weekStart time.Weekday) []protocol.Item {
	var items []protocol.Item
	query = strings.TrimSpace(query)
	// The filter is the text without its date words: "milk friday" is
	// about to add a task, and still finds the milk one already there.
	text, due := Split(query, now)
	q := match.New(text)
	var best match.Best[Task]
	done := 0
	for _, t := range ts {
		if t.Done {
			done++
		} else if score, ok := q.Score(t.Text); ok {
			best.Add(t, score)
		}
	}
	// Best first here, and arrange keeps that order inside one day.
	open := best.Rows()
	dues := make([]string, len(open))
	for i, t := range open {
		dues[i] = t.Due
	}
	if query != "" {
		items = append(items, protocol.Item{ID: "add" + sep + query, Kind: "toggle", Icon: iconAdd, Title: "Add task: " + text, Subtitle: describe(due, now)})
	} else if len(open) == 0 {
		items = append(items, protocol.Item{ID: "add" + sep, Kind: "toggle", Icon: iconAdd, Title: "Type a task, then Enter", Subtitle: "end it with today, tomorrow, a weekday or a date to give it one"})
	}
	order, groups := arrange(dues, now, weekStart)
	for k, i := range order {
		if k == 0 || groups[k] != groups[k-1] {
			n := 1
			for n+k < len(groups) && groups[n+k] == groups[k] {
				n++
			}
			items = append(items, protocol.Item{ID: groupID + sep + groups[k].String(), Kind: "group", Title: groups[k].String(), Subtitle: count(n, "task")})
		}
		t := open[i]
		it := protocol.Item{ID: "t" + sep + t.Raw, Kind: "toggle", Icon: iconOpen, Title: t.Text}
		if t.Due != "" {
			it.Subtitle = Label(t.Due, now)
		}
		items = append(items, it)
	}
	if done > 0 {
		// A switch, not a group: it is there whatever is typed, and says
		// how many it holds, not how many match.
		items = append(items, protocol.Item{ID: doneID, Kind: "view", Icon: iconDone, Title: "Done", Subtitle: count(done, "task") + ", Enter shows them"})
	}
	return items
}

// doneRows is the view the Done row opens: Back to open first, then the
// done tasks that match every word typed, the best match first and then
// the most recently done.
// at is when each was ticked, by its line (see loadDone); a task ticked
// in an editor has no time, and comes after the ones that do, the ones
// lower in the file first, since those were added later.
//
// Back to open is kind "refresh": Enter runs it, which does nothing, and
// goes back to the pane below, the open tasks. A done task is a toggle
// like an open one, so Enter opens it again and the view stays.
func doneRows(ts []Task, query string, at map[string]int64) []protocol.Item {
	q := match.New(query)
	open := 0
	var done []Task
	for i := len(ts) - 1; i >= 0; i-- {
		if !ts[i].Done {
			open++
		} else {
			done = append(done, ts[i])
		}
	}
	sort.SliceStable(done, func(i, j int) bool { return at[done[i].Raw] > at[done[j].Raw] })
	done = match.Rank(done, func(t Task) (int, bool) { return q.Score(t.Text) })
	items := []protocol.Item{{ID: openID, Kind: "refresh", Icon: iconBack, Title: "Back to open", Subtitle: count(open, "open task")}}
	for _, t := range done {
		// The protocol has no style field to dim a done task with; the
		// ticked box and the word do that.
		items = append(items, protocol.Item{ID: "t" + sep + t.Raw, Kind: "toggle", Icon: iconDone, Title: t.Text, Subtitle: "done"})
	}
	return items
}

// count is "1 task", "3 tasks".
func count(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return fmt.Sprintf("%d %ss", n, what)
}

// furniture says whether id is a row that is not a task: a header, the
// Done row, Back to open. It has no actions, and nothing to run.
func furniture(id string) bool {
	kind, _, _ := strings.Cut(id, sep)
	return kind == groupID || kind == doneID || kind == openID
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
	switch kind, name, _ := strings.Cut(id, sep); kind {
	case groupID:
		fmt.Println(name)
		fmt.Println()
		fmt.Println("The open tasks due then, the soonest first.")
		return nil
	case doneID:
		fmt.Println("The tasks you have ticked, the most recently done first.")
		fmt.Println()
		fmt.Println("Enter shows them. Back to open, or Esc, comes back here.")
		return nil
	case openID:
		fmt.Println("Enter goes back to the open tasks. Esc does too.")
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
// yet to act on, and neither have a header, Done, or Back to open.
func actions(id string) error {
	edit := protocol.Item{ID: "edit", Kind: "terminal", Icon: iconOpen, Title: "Edit the list", Subtitle: "the whole file, in " + strings.Join(settings.Editor(), " ")}
	if id == viewID {
		return protocol.Write(os.Stdout, []protocol.Item{edit})
	}
	if furniture(id) {
		return nil
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
	if furniture(id) {
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
		return addTo(f, text, now)
	}
	if id == viewID {
		return nil
	}
	ticked := false
	switch action {
	case "":
		f.SetDone(t, !t.Done)
		ticked = !t.Done
	case "done":
		f.SetDone(t, true)
		ticked = true
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
	if err := f.Save(); err != nil {
		return err
	}
	if ticked {
		// When, for the done view's order. The task's line as it is now.
		return noteDone(donePath(), f, f.Lines[t.Line], now)
	}
	return nil
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
