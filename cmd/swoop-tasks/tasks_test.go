package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
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

// titles are the rows' titles, a header's in brackets, so a test can say
// a whole view in one line.
func titles(rs []protocol.Item) string {
	var out []string
	for _, r := range rs {
		if r.Kind == "group" {
			out = append(out, "["+r.Title+"]")
		} else {
			out = append(out, r.Title)
		}
	}
	return strings.Join(out, ",")
}

func TestGroupOf(t *testing.T) {
	at := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 10, 30, 0, 0, time.Local) }
	for _, c := range []struct {
		name string
		now  time.Time
		week time.Weekday
		due  string
		want Group
	}{
		// Mid month, Monday 14 September 2026, the week from Monday.
		{"yesterday", at(2026, 9, 14), time.Monday, "2026-09-13", PastDue},
		{"long ago", at(2026, 9, 14), time.Monday, "2025-01-01", PastDue},
		{"today", at(2026, 9, 14), time.Monday, "2026-09-14", Today},
		{"tomorrow", at(2026, 9, 14), time.Monday, "2026-09-15", Tomorrow},
		{"friday", at(2026, 9, 14), time.Monday, "2026-09-18", ThisWeek},
		{"the week's last day", at(2026, 9, 14), time.Monday, "2026-09-20", ThisWeek},
		{"the next week's first", at(2026, 9, 14), time.Monday, "2026-09-21", ThisMonth},
		{"the 30th", at(2026, 9, 14), time.Monday, "2026-09-30", ThisMonth},
		{"the month after", at(2026, 9, 14), time.Monday, "2026-10-01", Later},
		{"no day", at(2026, 9, 14), time.Monday, "", Unscheduled},
		{"not a date", at(2026, 9, 14), time.Monday, "2026-13-01", Unscheduled},

		// A month boundary: Wednesday 30 September. Tomorrow is in October,
		// the week runs into October, and nothing is left of this month.
		{"month end: yesterday", at(2026, 9, 30), time.Monday, "2026-09-29", PastDue},
		{"month end: tomorrow is the 1st", at(2026, 9, 30), time.Monday, "2026-10-01", Tomorrow},
		{"month end: friday, in the next month", at(2026, 9, 30), time.Monday, "2026-10-02", ThisWeek},
		{"month end: sunday ends the week", at(2026, 9, 30), time.Monday, "2026-10-04", ThisWeek},
		{"month end: past the week is past the month", at(2026, 9, 30), time.Monday, "2026-10-05", Later},
		{"month end: the 30th of the next", at(2026, 9, 30), time.Monday, "2026-10-30", Later},
		// A year boundary is a month boundary too: Thursday 31 December.
		{"year end: tomorrow", at(2026, 12, 31), time.Monday, "2027-01-01", Tomorrow},
		{"year end: sunday", at(2026, 12, 31), time.Monday, "2027-01-03", ThisWeek},
		{"year end: monday", at(2026, 12, 31), time.Monday, "2027-01-04", Later},

		// A week boundary: Sunday 27 September is the last day of a week
		// from Monday, so tomorrow is next week and This week is empty.
		{"week end: tomorrow", at(2026, 9, 27), time.Monday, "2026-09-28", Tomorrow},
		{"week end: the day after", at(2026, 9, 27), time.Monday, "2026-09-29", ThisMonth},
		{"week end: wednesday", at(2026, 9, 27), time.Monday, "2026-09-30", ThisMonth},
		// The same Sunday is the first day of a week from Sunday, which
		// then runs to Saturday 3 October, across the month.
		{"sunday week: the day after tomorrow", at(2026, 9, 27), time.Sunday, "2026-09-29", ThisWeek},
		{"sunday week: saturday, in the next month", at(2026, 9, 27), time.Sunday, "2026-10-03", ThisWeek},
		{"sunday week: the next sunday", at(2026, 9, 27), time.Sunday, "2026-10-04", Later},
		// Saturday 26 September: the last day of a week from Sunday, the
		// day before the last of one from Monday.
		{"saturday, sunday week: monday", at(2026, 9, 26), time.Sunday, "2026-09-28", ThisMonth},
		{"saturday, monday week: sunday is tomorrow", at(2026, 9, 26), time.Monday, "2026-09-27", Tomorrow},
		{"saturday, monday week: monday", at(2026, 9, 26), time.Monday, "2026-09-28", ThisMonth},
		// Friday, in a week from Monday: Sunday is still this week.
		{"friday: sunday", at(2026, 9, 25), time.Monday, "2026-09-27", ThisWeek},
		{"friday, sunday week: sunday", at(2026, 9, 25), time.Sunday, "2026-09-27", ThisMonth},
	} {
		if got := GroupOf(c.due, c.now, c.week); got != c.want {
			t.Errorf("%s: GroupOf(%q) = %s, want %s", c.name, c.due, got, c.want)
		}
	}
}

// sixGroups is one task for each header, out of order in the file, and a
// done one. On Monday 14 September 2026 they are due yesterday, today,
// tomorrow, Friday, the 30th, and never.
var sixGroups = []Task{
	{Raw: "- [ ] none", Text: "none"},
	{Raw: "- [ ] the 30th due: 2026-09-30", Text: "the 30th", Due: "2026-09-30"},
	{Raw: "- [x] ticked", Text: "ticked", Done: true},
	{Raw: "- [ ] friday due: 2026-09-18", Text: "friday", Due: "2026-09-18"},
	{Raw: "- [ ] tomorrow due: 2026-09-15", Text: "tomorrow", Due: "2026-09-15"},
	{Raw: "- [ ] today due: 2026-09-14", Text: "today", Due: "2026-09-14"},
	{Raw: "- [ ] yesterday due: 2026-09-13", Text: "yesterday", Due: "2026-09-13"},
}

var midMonth = time.Date(2026, 9, 14, 10, 30, 0, 0, time.Local)

func TestRowsGroupsInOrder(t *testing.T) {
	got := titles(rows(sixGroups, "", midMonth, time.Monday))
	want := "[Past due],yesterday,[Today],today,[Tomorrow],tomorrow,[This week],friday,[This month],the 30th,[Unscheduled],none,Done"
	if got != want {
		t.Fatalf("the view:\n%s\nwant:\n%s", got, want)
	}
	for _, r := range rows(sixGroups, "", midMonth, time.Monday) {
		switch {
		case r.Kind == "group" && !strings.HasPrefix(r.ID, "group"+sep):
			t.Fatalf("a header's id: %+v", r)
		case r.ID == "done" && r.Kind != "view":
			t.Fatalf("Done opens a view: %+v", r)
		case strings.HasPrefix(r.ID, "t"+sep) && r.Kind != "toggle":
			t.Fatalf("a task row must be a toggle, so the view stays: %+v", r)
		}
	}
	// An empty group has no header: on the 30th nothing is left of the
	// month, and what was due before it is past due.
	got = titles(rows(sixGroups, "", time.Date(2026, 9, 30, 9, 0, 0, 0, time.Local), time.Monday))
	want = "[Past due],yesterday,today,tomorrow,friday,[Today],the 30th,[Unscheduled],none,Done"
	if got != want {
		t.Fatalf("on the 30th:\n%s\nwant:\n%s", got, want)
	}
}

func TestRowsOrderInsideAGroup(t *testing.T) {
	ts := []Task{
		{Raw: "a", Text: "sunday", Due: "2026-09-20"},
		{Raw: "b", Text: "thursday", Due: "2026-09-17"},
		{Raw: "c", Text: "plain one"},
		{Raw: "d", Text: "also thursday", Due: "2026-09-17"},
		{Raw: "e", Text: "plain two"},
	}
	// The soonest first, then the order in the file.
	got := titles(rows(ts, "", midMonth, time.Monday))
	want := "[This week],thursday,also thursday,sunday,[Unscheduled],plain one,plain two"
	if got != want {
		t.Fatalf("inside a group:\n%s\nwant:\n%s", got, want)
	}
	// From Sunday the week ends on Saturday the 19th.
	got = titles(rows(ts, "", midMonth, time.Sunday))
	want = "[This week],thursday,also thursday,[This month],sunday,[Unscheduled],plain one,plain two"
	if got != want {
		t.Fatalf("a week from Sunday:\n%s\nwant:\n%s", got, want)
	}
}

func TestRowsFilterHidesEmptyGroups(t *testing.T) {
	got := titles(rows(sixGroups, "to", midMonth, time.Monday))
	want := "Add task: to,[Today],today,[Tomorrow],tomorrow,Done"
	if got != want {
		t.Fatalf("filtered:\n%s\nwant:\n%s", got, want)
	}
	rs := rows(sixGroups, "to", midMonth, time.Monday)
	if rs[1].Subtitle != "1 task" {
		t.Fatalf("a header counts what it shows: %+v", rs[1])
	}
	// A done task is not in this view, whatever is typed.
	if got := titles(rows(sixGroups, "ticked", midMonth, time.Monday)); got != "Add task: ticked,Done" {
		t.Fatalf("a done task matched in the open view: %s", got)
	}
}

func TestRows(t *testing.T) {
	ts := []Task{
		{Raw: "- [ ] buy milk due: 2026-09-29", Text: "buy milk", Due: "2026-09-29"},
		{Raw: "- [x] call the bank", Text: "call the bank", Done: true},
	}
	rs := rows(ts, "", monday, time.Monday)
	if titles(rs) != "[Tomorrow],buy milk,Done" || rs[1].Subtitle != "tomorrow, Tue 29 Sep" || rs[2].Subtitle != "1 task, Enter shows them" {
		t.Fatalf("nothing typed: %+v", rs)
	}
	rs = rows(ts, "Milk friday", monday, time.Monday)
	if titles(rs) != "Add task: Milk,[Tomorrow],buy milk,Done" || rs[0].Subtitle != "due Friday" || rs[0].ID != "add"+sep+"Milk friday" {
		t.Fatalf("typed: %+v", rs)
	}
	rs = rows(nil, "", monday, time.Monday)
	if len(rs) != 1 || rs[0].ID != "add"+sep {
		t.Fatalf("an empty list shows the hint: %+v", rs)
	}
	// Nothing open: the hint, and the way to the done ones.
	if got := titles(rows(ts[1:], "", monday, time.Monday)); got != "Type a task, then Enter,Done" {
		t.Fatalf("nothing open: %s", got)
	}
}

func TestDoneRows(t *testing.T) {
	ts := []Task{
		{Raw: "- [x] first in the file", Text: "first in the file", Done: true},
		{Raw: "- [ ] open one", Text: "open one"},
		{Raw: "- [x] ticked early", Text: "ticked early", Done: true},
		{Raw: "- [x] ticked late", Text: "ticked late", Done: true},
		{Raw: "- [x] last in the file", Text: "last in the file", Done: true},
	}
	at := map[string]int64{"- [x] ticked early": 100, "- [x] ticked late": 200}
	rs := doneRows(ts, "", at)
	// The most recently done first; the ones with no time after, the
	// lowest in the file first.
	if got := titles(rs); got != "Back to open,ticked late,ticked early,last in the file,first in the file" {
		t.Fatalf("the done view: %s", got)
	}
	if rs[0].ID != "open" || rs[0].Kind != "refresh" || rs[0].Subtitle != "1 open task" {
		t.Fatalf("Back to open goes back to the pane below: %+v", rs[0])
	}
	for _, r := range rs[1:] {
		if r.Kind != "toggle" || r.Subtitle != "done" {
			t.Fatalf("a done task: %+v", r)
		}
	}
	if got := titles(doneRows(ts, "LATE", at)); got != "Back to open,ticked late" {
		t.Fatalf("filtered: %s", got)
	}
	if got := titles(doneRows(ts, "open", at)); got != "Back to open" {
		t.Fatalf("an open task in the done view: %s", got)
	}
}

func TestGroupLines(t *testing.T) {
	in := "\tb\tno day\n2026-09-30\ta\tthe 30th\n2026-09-14\tc\ttoday two\n2026-09-13\td\tlate\n2026-09-14\te\ttoday one\n"
	var out strings.Builder
	if err := groupLines(strings.NewReader(in), &out, midMonth, time.Monday); err != nil {
		t.Fatal(err)
	}
	want := "Past due\t2026-09-13\td\tlate\n" +
		"Today\t2026-09-14\tc\ttoday two\n" +
		"Today\t2026-09-14\te\ttoday one\n" +
		"This month\t2026-09-30\ta\tthe 30th\n" +
		"Unscheduled\t\tb\tno day\n"
	if out.String() != want {
		t.Fatalf("group:\n%s\nwant:\n%s", out.String(), want)
	}
}

// useDir points the file at a temp directory for one test.
func useDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_DATA_HOME", d)
	// The log of when each task was ticked goes here.
	t.Setenv("XDG_STATE_HOME", filepath.Join(d, "state"))
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

func TestTickingIsLoggedForTheDoneOrder(t *testing.T) {
	useDir(t)
	for _, text := range []string{"one", "two", "three"} {
		if err := addText(text, monday); err != nil {
			t.Fatal(err)
		}
	}
	id := func(box, text string) string { return "t" + sep + "- [" + box + "] " + text }
	// Ticked in this order: two, then one, a minute apart.
	if err := run(id(" ", "two"), "", monday); err != nil {
		t.Fatal(err)
	}
	if err := run(id(" ", "one"), "done", monday.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	view := func() string {
		f, err := load(path())
		if err != nil {
			t.Fatal(err)
		}
		return titles(doneRows(f.Tasks(), "", loadDone(donePath())))
	}
	if got := view(); got != "Back to open,one,two" {
		t.Fatalf("the most recently done first: %s", got)
	}
	// Opened again, a task leaves the log, and the view.
	if err := run(id("x", "one"), "", monday.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := run(id(" ", "three"), "", monday.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got := view(); got != "Back to open,three,two" {
		t.Fatalf("after an undo: %s", got)
	}
	if log := read(t, donePath()); strings.Contains(log, "one") || strings.Count(log, "\n") != 2 {
		t.Fatalf("the log keeps only the done tasks:\n%s", log)
	}
}

func TestHeadersAndSwitchesRunNothing(t *testing.T) {
	p := useDir(t)
	if err := addText("water the plants", monday); err != nil {
		t.Fatal(err)
	}
	before := read(t, p)
	for _, id := range []string{"group" + sep + "Today", "done", "open"} {
		if err := run(id, "", monday); err != nil {
			t.Fatalf("run %q: %v", id, err)
		}
		if err := actions(id); err != nil {
			t.Fatalf("actions %q: %v", id, err)
		}
	}
	if got := read(t, p); got != before {
		t.Fatalf("a header changed the file:\n%s", got)
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
