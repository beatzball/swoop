// swoop-usage is the log of what the launcher opened, and the Stats pane
// over it. It speaks the extension contract, so the launcher routes to
// it like any other, and two verbs more for the tools that write.
//
//	swoop-usage record <id> [kind] [title]   one open, appended; swoop-run calls it
//	swoop-usage recent [n]                    the n most recent ids, newest first
//	swoop-usage clear                         remove the log
//	swoop-usage list                          the Stats row
//	swoop-usage view stats [q]                what was opened, most used first
//	swoop-usage preview <id>                  count, first and last, a fortnight by day
//	swoop-usage run <id> [action]             open it again; clear
//	swoop-usage actions <id>                  Clear the log
//
// The log is ~/.local/state/swoop/usage.jsonl. See internal/usage.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/usage"
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
	var err error
	switch os.Args[1] {
	case "record":
		err = usage.Record(arg(2), arg(3), arg(4))
	case "recent":
		n := 5
		if v, e := strconv.Atoi(arg(2)); e == nil && v > 0 {
			n = v
		}
		entries, e := usage.Load()
		if e != nil {
			err = e
			break
		}
		for _, id := range usage.Recent(entries, n) {
			fmt.Println(id)
		}
	case "clear":
		err = usage.Clear()
	case "list":
		err = protocol.Write(os.Stdout, []protocol.Item{{ID: "stats", Kind: "view", Icon: "󰄨", Title: "Stats", Subtitle: "what you open most, and when"}})
	case "view":
		err = view(strings.TrimSpace(arg(3)))
	case "preview":
		err = preview(arg(2))
	case "actions":
		if arg(2) != "" {
			err = protocol.Write(os.Stdout, []protocol.Item{{ID: "clear", Kind: "refresh", Icon: "", Title: "Clear the usage log", Subtitle: "counts start over"}})
		}
	case "run":
		err = run(arg(2), arg(3))
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-usage:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-usage record <id> [kind] [title] | recent [n] | clear | list | view stats [q] | preview <id> | run <id> [action] | actions <id>")
	os.Exit(2)
}

// view lists what was opened, most used first, the count and the last
// time in the subtitle. The id is the opened thing's own id, so Enter
// opens it again and the preview is its usage. The query filters titles.
func view(query string) error {
	entries, err := usage.Load()
	if err != nil {
		return err
	}
	var items []protocol.Item
	for _, s := range usage.Stats(entries) {
		title := s.Title
		if title == "" {
			title = s.ID
		}
		if query != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
			continue
		}
		items = append(items, protocol.Item{ID: s.ID, Kind: "stat", Icon: iconFor(s.Kind), Title: title, Subtitle: fmt.Sprintf("%s · %s", times(s.Count), ago(s.Last))})
	}
	if len(items) == 0 {
		items = append(items, protocol.Item{ID: "none", Kind: "text", Icon: "", Title: "Nothing opened yet", Subtitle: "every Enter counts from now on"})
	}
	return protocol.Write(os.Stdout, items)
}

func iconFor(kind string) string {
	switch kind {
	case "app":
		return ""
	case "ai":
		return "󰭹"
	default:
		return ""
	}
}

// preview is one id's story: count, first and last, and a fortnight of
// days as bars. For the Stats row itself, the totals.
func preview(id string) error {
	entries, err := usage.Load()
	if err != nil {
		return err
	}
	if id == "stats" || id == "" || id == "none" {
		days := byDay(entries, 14, "")
		fmt.Printf("\x1b[1m%s\x1b[22m opened, %s different things, since %s\n\n", times(len(entries)), strconv.Itoa(len(usage.Stats(entries))), firstDay(entries))
		fmt.Print(bars(days))
		fmt.Printf("\n\x1b[2m%s\x1b[22m\n", usage.Path())
		return nil
	}
	for _, s := range usage.Stats(entries) {
		if s.ID != id {
			continue
		}
		fmt.Printf("\x1b[1m%s\x1b[22m\n\n%s, first %s, last %s\n\n", s.Title, times(s.Count), ago(s.First), ago(s.Last))
		fmt.Print(bars(byDay(entries, 14, id)))
		return nil
	}
	fmt.Println("Not opened yet.")
	return nil
}

func run(id, action string) error {
	if action == "clear" {
		return usage.Clear()
	}
	return nil
}

// byDay counts the opens per day for the last n days, oldest first, for
// one id or for all.
func byDay(entries []usage.Entry, n int, id string) []int {
	counts := make([]int, n)
	today := time.Now().Truncate(24 * time.Hour)
	for _, e := range entries {
		if id != "" && e.ID != id {
			continue
		}
		d := int(today.Sub(e.Time.Truncate(24*time.Hour)).Hours() / 24)
		if d >= 0 && d < n {
			counts[n-1-d]++
		}
	}
	return counts
}

// bars draws a day per line, the newest last, a bar of blocks scaled to
// the busiest day.
func bars(days []int) string {
	max := 0
	for _, c := range days {
		if c > max {
			max = c
		}
	}
	var b strings.Builder
	b.WriteString("\x1b[2mlast 14 days\x1b[22m\n")
	for i, c := range days {
		day := time.Now().AddDate(0, 0, -(len(days) - 1 - i))
		w := 0
		if max > 0 {
			w = c * 20 / max
		}
		if c > 0 && w == 0 {
			w = 1
		}
		fmt.Fprintf(&b, "%s %s %d\n", day.Format("Mon 02"), strings.Repeat("█", w), c)
	}
	return b.String()
}

func times(n int) string {
	if n == 1 {
		return "once"
	}
	return strconv.Itoa(n) + " times"
}

func ago(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func firstDay(entries []usage.Entry) string {
	if len(entries) == 0 {
		return "today"
	}
	return entries[0].Time.Format("Jan 2")
}
