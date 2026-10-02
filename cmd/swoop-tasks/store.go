// The tasks file. One markdown checklist, ~/.local/share/<name>/tasks.md:
//
//   - [ ] buy milk due: 2026-09-29
//   - [x] call the bank
//
// A task is a line that starts with "- [ ]" (open) or "- [x]" (done),
// then its text, then an optional "due:" and a date. Every other line, a
// heading, a note, a blank, is kept as it is: the file is the user's, and
// The tool only ever changes the one line it was asked to change.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/tool"
)

// Task is one checklist line.
type Task struct {
	Line int    // index into the file's lines
	Raw  string // the line as it is in the file; the row's id carries it
	Done bool
	Text string
	Due  string // 2026-09-29, or ""
}

var (
	// taskLine is a checklist item: a list marker, a box, the rest. Any
	// of the three markdown list markers, and an X in either case, so a
	// file written by another editor reads the same.
	taskLine = regexp.MustCompile(`^\s*[-*+] \[([ xX])\] ?(.*)$`)
	// dueTail is the due date at the end of a task's text.
	dueTail = regexp.MustCompile(`^(.*?)\s*\bdue:\s*(\d{4}-\d{2}-\d{2})\s*$`)
)

// parseLine reads one line as a task, if it is one.
func parseLine(i int, line string) (Task, bool) {
	m := taskLine.FindStringSubmatch(line)
	if m == nil {
		return Task{}, false
	}
	t := Task{Line: i, Raw: line, Done: m[1] != " ", Text: strings.TrimSpace(m[2])}
	if d := dueTail.FindStringSubmatch(t.Text); d != nil {
		t.Text, t.Due = strings.TrimSpace(d[1]), d[2]
	}
	return t, true
}

// format writes a task as its line.
func format(done bool, text, due string) string {
	box := "[ ]"
	if done {
		box = "[x]"
	}
	line := "- " + box + " " + oneLine(text)
	if due != "" {
		line += " due: " + due
	}
	return line
}

// oneLine keeps a task on its line: a newline typed or pasted into the
// bar would split it in two.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// path is the file, ~/.local/share/<name>/tasks.md, or under
// XDG_DATA_HOME when that is set, like the clipboard history.
func path() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, tool.Name(), "tasks.md")
}

// File is the tasks file in memory, line by line.
type File struct {
	Path  string
	Lines []string
}

// load reads the file. No file is no tasks, not an error: it is written
// on the first add.
func load(p string) (*File, error) {
	f := &File{Path: p}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text != "" {
		f.Lines = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	}
	return f, nil
}

// Tasks are the file's checklist lines, in file order.
func (f *File) Tasks() []Task {
	var ts []Task
	for i, l := range f.Lines {
		if t, ok := parseLine(i, l); ok {
			ts = append(ts, t)
		}
	}
	return ts
}

// find is the task whose line is raw, the first one when two lines are
// the same, which is harmless: they are the same task.
func (f *File) find(raw string) (Task, bool) {
	for _, t := range f.Tasks() {
		if t.Raw == raw {
			return t, true
		}
	}
	return Task{}, false
}

// Add appends a task. A new file starts with a heading, so it reads as
// what it is when opened in an editor.
func (f *File) Add(text, due string) {
	if len(f.Lines) == 0 {
		f.Lines = []string{"# Tasks", ""}
	}
	f.Lines = append(f.Lines, format(false, text, due))
}

// SetDone ticks a task, or unticks it.
func (f *File) SetDone(t Task, done bool) {
	f.Lines[t.Line] = format(done, t.Text, t.Due)
}

// Delete takes a task's line out.
func (f *File) Delete(t Task) {
	f.Lines = append(f.Lines[:t.Line], f.Lines[t.Line+1:]...)
}

// Save writes the file whole, through a temporary file and a rename, so
// a crash mid-write leaves the old file as it was.
func (f *File) Save() error {
	if f.Path == "" {
		return errors.New("no data directory")
	}
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o700); err != nil {
		return err
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(f.Lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.Path)
}

// donePath is the log of when each task was ticked: a time and the
// task's line, tab-separated, in ~/.local/state/<name>/tasks-done.tsv, or
// under XDG_STATE_HOME. The done view reads it to put the most recently
// done first. It is beside the tasks file, not in it: the checklist stays
// a plain one that any editor can change, and losing the log loses only
// an order.
func donePath() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, tool.Name(), "tasks-done.tsv")
}

// loadDone reads the log: when each line was ticked, in nanoseconds. No
// log, or one that cannot be read, is no times, and the view still has
// an order.
func loadDone(p string) map[string]int64 {
	at := map[string]int64{}
	data, err := os.ReadFile(p)
	if err != nil {
		return at
	}
	for _, l := range strings.Split(string(data), "\n") {
		when, line, ok := strings.Cut(l, "\t")
		if n, err := strconv.ParseInt(when, 10, 64); ok && err == nil {
			at[line] = n
		}
	}
	return at
}

// noteDone writes that line was ticked at now. The log is written whole,
// with only the lines that are done tasks in the file now, so a task
// opened again, deleted or reworded does not stay in it for ever.
func noteDone(p string, f *File, line string, now time.Time) error {
	if p == "" {
		return errors.New("no state directory")
	}
	at := loadDone(p)
	at[line] = now.UnixNano()
	var out strings.Builder
	seen := map[string]bool{}
	for _, t := range f.Tasks() {
		if n, ok := at[t.Raw]; ok && t.Done && !seen[t.Raw] {
			seen[t.Raw] = true
			out.WriteString(strconv.FormatInt(n, 10) + "\t" + t.Raw + "\n")
		}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(out.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
