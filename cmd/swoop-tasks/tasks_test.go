package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// monday is the clock the tests read: Monday 28 September 2026, mid
// morning.
var monday = time.Date(2026, 9, 28, 10, 30, 0, 0, time.Local)

func TestSplitReadsTheDateWordsAtTheEnd(t *testing.T) {
	for _, c := range []struct{ in, text, due string }{
		{"buy milk tomorrow", "buy milk", "2026-09-29"},
		{"buy milk Tomorrow", "buy milk", "2026-09-29"},
		{"call mum today", "call mum", "2026-09-28"},
		{"pack tonight", "pack", "2026-09-28"},
		{"send the report friday", "send the report", "2026-10-02"},
		{"send the report by fri", "send the report", "2026-10-02"},
		{"dentist on tuesday", "dentist", "2026-09-29"},
		{"standup next wed", "standup", "2026-09-30"},
		// A weekday is the next one to come: on a Monday, "monday" is a
		// week away, not today.
		{"plan the week monday", "plan the week", "2026-10-05"},
		{"sunday roast sunday", "sunday roast", "2026-10-04"},
		{"renew passport 2026-10-01", "renew passport", "2026-10-01"},
		{"rent due 2027-01-01", "rent", "2027-01-01"},
		// Only the end is read.
		{"call Friday's venue", "call Friday's venue", ""},
		{"friday drinks", "friday drinks", ""},
		{"buy milk", "buy milk", ""},
		// Something has to be left to do.
		{"tomorrow", "tomorrow", ""},
		{"  spaced   out   tomorrow ", "spaced out", "2026-09-29"},
		// A connector alone before the date is the task, not a connector.
		{"by tomorrow", "by", "2026-09-29"},
		{"not a date 2026-13-01", "not a date 2026-13-01", ""},
	} {
		text, due := Split(c.in, monday)
		if text != c.text || due != c.due {
			t.Errorf("Split(%q) = %q, %q; want %q, %q", c.in, text, due, c.text, c.due)
		}
	}
}

func TestSplitAcrossAMonthAndAYear(t *testing.T) {
	nye := time.Date(2026, 12, 31, 23, 0, 0, 0, time.Local)
	if _, due := Split("party tomorrow", nye); due != "2027-01-01" {
		t.Fatalf("tomorrow on new year's eve: %q", due)
	}
}

func TestLabel(t *testing.T) {
	for _, c := range []struct{ due, want string }{
		{"2026-09-27", "overdue, Sun 27 Sep"},
		{"2026-09-28", "today"},
		{"2026-09-29", "tomorrow, Tue 29 Sep"},
		{"2026-10-02", "Friday"},
		{"2026-10-05", "Mon 5 Oct"},
		{"2027-01-04", "Mon 4 Jan 2027"},
	} {
		if got := Label(c.due, monday); got != c.want {
			t.Errorf("Label(%q) = %q, want %q", c.due, got, c.want)
		}
	}
}

func TestParseLine(t *testing.T) {
	for _, c := range []struct {
		line string
		ok   bool
		want Task
	}{
		{"- [ ] buy milk due: 2026-09-29", true, Task{Text: "buy milk", Due: "2026-09-29"}},
		{"- [x] call the bank", true, Task{Done: true, Text: "call the bank"}},
		{"* [X] other editor", true, Task{Done: true, Text: "other editor"}},
		{"  - [ ] indented", true, Task{Text: "indented"}},
		{"- [ ] due: 2026-09-29", true, Task{Due: "2026-09-29"}},
		{"# Tasks", false, Task{}},
		{"- a plain list item", false, Task{}},
		{"", false, Task{}},
	} {
		got, ok := parseLine(0, c.line)
		c.want.Raw = c.line
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseLine(%q) = %+v, %v; want %+v, %v", c.line, got, ok, c.want, c.ok)
		}
	}
}

func TestOrderPutsDueFirstThenOpenThenDone(t *testing.T) {
	ts := []Task{
		{Line: 0, Text: "no date"},
		{Line: 1, Text: "done", Done: true},
		{Line: 2, Text: "friday", Due: "2026-10-02"},
		{Line: 3, Text: "late", Due: "2026-09-20"},
		{Line: 4, Text: "also no date"},
		{Line: 5, Text: "done late", Done: true, Due: "2026-09-01"},
	}
	var got []string
	for _, t := range order(ts, monday) {
		got = append(got, t.Text)
	}
	want := "late,friday,no date,also no date,done,done late"
	if strings.Join(got, ",") != want {
		t.Fatalf("order: %s, want %s", strings.Join(got, ","), want)
	}
}

func TestRows(t *testing.T) {
	ts := []Task{
		{Raw: "- [ ] buy milk due: 2026-09-29", Text: "buy milk", Due: "2026-09-29"},
		{Raw: "- [x] call the bank", Text: "call the bank", Done: true},
	}
	rs := rows(ts, "", monday)
	if len(rs) != 2 || rs[0].Title != "buy milk" || rs[0].Subtitle != "tomorrow, Tue 29 Sep" || rs[1].Subtitle != "done" {
		t.Fatalf("nothing typed: %+v", rs)
	}
	for _, r := range rs {
		if r.Kind != "toggle" {
			t.Fatalf("a task row must be a toggle, so the view stays: %+v", r)
		}
	}
	rs = rows(ts, "Milk friday", monday)
	if len(rs) != 2 || rs[0].Title != "Add task: Milk" || rs[0].Subtitle != "due Friday" || rs[0].ID != "add"+sep+"Milk friday" {
		t.Fatalf("typed: %+v", rs)
	}
	rs = rows(ts, "milk", monday)
	if len(rs) != 2 || rs[1].Title != "buy milk" {
		t.Fatalf("a filter: %+v", rs)
	}
	rs = rows(nil, "", monday)
	if len(rs) != 1 || rs[0].ID != "add"+sep {
		t.Fatalf("an empty list shows the hint: %+v", rs)
	}
}

// useDir points the file at a temp directory for one test.
func useDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_DATA_HOME", d)
	return filepath.Join(d, "swoop", "tasks.md")
}

func TestAddTickUndoDelete(t *testing.T) {
	p := useDir(t)
	// A line of the user's own, which must survive every change.
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Mine\n\nsome notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run("add"+sep+"buy milk tomorrow", "", monday); err != nil {
		t.Fatal(err)
	}
	if err := run("add"+sep+"   ", "", monday); err != nil {
		t.Fatal(err)
	}
	want := "# Mine\n\nsome notes\n- [ ] buy milk due: 2026-09-29\n"
	if got := read(t, p); got != want {
		t.Fatalf("after add:\n%s\nwant:\n%s", got, want)
	}
	id := "t" + sep + "- [ ] buy milk due: 2026-09-29"
	if err := run(id, "", monday); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); !strings.Contains(got, "- [x] buy milk due: 2026-09-29\n") {
		t.Fatalf("Enter did not tick it:\n%s", got)
	}
	done := "t" + sep + "- [x] buy milk due: 2026-09-29"
	if err := run(done, "undo", monday); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); got != want {
		t.Fatalf("Undo did not open it again:\n%s", got)
	}
	if err := run(done, "", monday); err == nil {
		t.Fatal("an id whose line is gone must fail, not tick another")
	}
	if err := run(id, "delete", monday); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); got != "# Mine\n\nsome notes\n" {
		t.Fatalf("after delete:\n%s", got)
	}
}

func TestFirstAddStartsTheFile(t *testing.T) {
	p := useDir(t)
	if err := addText("water the plants", monday); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); got != "# Tasks\n\n- [ ] water the plants\n" {
		t.Fatalf("new file:\n%s", got)
	}
}

func TestSummary(t *testing.T) {
	ts := []Task{{Text: "a", Due: "2026-09-28"}, {Text: "b"}, {Text: "c", Done: true}}
	if got := summary(ts, monday); got != "2 open, 1 due today or late" {
		t.Fatalf("summary: %q", got)
	}
	if got := summary(nil, monday); got != "a checklist; nothing open" {
		t.Fatalf("empty: %q", got)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
