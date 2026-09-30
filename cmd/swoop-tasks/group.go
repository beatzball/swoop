// Groups: the view's headers, Past due to Unscheduled, and which one a
// due day falls under. Like the date words, nothing here reads the clock
// or the settings file; the day and the week's first day are arguments,
// so the tests can pick both.
package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Group is a header of the view. Their order here is their order there.
type Group int

const (
	PastDue Group = iota
	Today
	Tomorrow
	// ThisWeek is the rest of the calendar week after tomorrow.
	ThisWeek
	// ThisMonth is the rest of the month after that week.
	ThisMonth
	// Later is a due day past the end of the month. Without it a task due
	// next month would have to sit under a header that is not true of it.
	Later
	Unscheduled
)

var groupNames = [...]string{"Past due", "Today", "Tomorrow", "This week", "This month", "Later", "Unscheduled"}

func (g Group) String() string { return groupNames[g] }

// GroupOf says which header a due day goes under on now's day, in a week
// that starts on weekStart. No due day, or one that is not a date, is
// Unscheduled. The week wins over the month: on Wednesday 30 September,
// Friday 2 October is This week, and nothing is left for This month.
func GroupOf(due string, now time.Time, weekStart time.Weekday) Group {
	n, ok := daysUntil(due, now)
	switch {
	case !ok:
		return Unscheduled
	case n < 0:
		return PastDue
	case n == 0:
		return Today
	case n == 1:
		return Tomorrow
	}
	// The week's last day is the one before its first; left is how many
	// days from today to it, 0 on that day itself.
	last := (int(weekStart) + 6) % 7
	left := (last - int(now.Weekday()) + 7) % 7
	if n <= left {
		return ThisWeek
	}
	t, _ := time.ParseInLocation(iso, due, now.Location())
	if t.Year() == now.Year() && t.Month() == now.Month() {
		return ThisMonth
	}
	return Later
}

// arrange puts due days in the view's order: by group, and inside a
// group the soonest first, then the order they came in. It returns the
// places in dues, and the group of each, in that order.
func arrange(dues []string, now time.Time, weekStart time.Weekday) (order []int, groups []Group) {
	of := make([]Group, len(dues))
	order = make([]int, len(dues))
	for i, d := range dues {
		of[i] = GroupOf(d, now, weekStart)
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		i, j := order[a], order[b]
		if of[i] != of[j] {
			return of[i] < of[j]
		}
		// Unscheduled holds no dates to compare, only their lack.
		return of[i] != Unscheduled && dues[i] < dues[j]
	})
	groups = make([]Group, len(order))
	for k, i := range order {
		groups[k] = of[i]
	}
	return order, groups
}

// groupLines is `swoop-tasks group`: lines in, each with its due day as
// the first tab-separated field, and the same lines out in the view's
// order, each with its group's name in front. It is for the reminders
// extension, so that both checklists draw the same headers.
func groupLines(r io.Reader, w io.Writer, now time.Time, weekStart time.Weekday) error {
	var lines, dues []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		due, _, _ := strings.Cut(sc.Text(), "\t")
		lines, dues = append(lines, sc.Text()), append(dues, due)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	bw := bufio.NewWriter(w)
	order, groups := arrange(dues, now, weekStart)
	for k, i := range order {
		fmt.Fprintf(bw, "%s\t%s\n", groups[k], lines[i])
	}
	return bw.Flush()
}
