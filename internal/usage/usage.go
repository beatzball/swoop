// Package usage keeps what the launcher opened: one line per open in
// ~/.local/state/swoop/usage.jsonl, appended by swoop-run and by an
// Ask AI send. From it come the Used recently group at the top of the
// root list and the Stats pane. It is the launcher's own data, in the
// user's state directory, and it never leaves the machine.
package usage

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
)

// Entry is one open.
type Entry struct {
	Time  time.Time `json:"t"`
	ID    string    `json:"id"`
	Kind  string    `json:"kind,omitempty"`
	Title string    `json:"title,omitempty"`
}

// Path is the log: $XDG_STATE_HOME/swoop/usage.jsonl, or
// ~/.local/state/swoop/usage.jsonl.
func Path() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "swoop", "usage.jsonl")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "swoop", "usage.jsonl")
}

// keep is how many lines the log holds before the oldest are dropped:
// enough for months of use, small enough to read on every launch.
const keep = 5000

// Record appends one open. A log that has grown past keep is cut back
// to its newest keep lines first. Nothing here is worth failing an open
// for, so a log that cannot be written is simply not written.
func Record(id, kind, title string) error {
	path := Path()
	if path == "" || id == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if st, err := os.Stat(path); err == nil && st.Size() > 512<<10 {
		compact(path)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(Entry{Time: time.Now(), ID: id, Kind: kind, Title: title})
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

func compact(path string) {
	entries, err := Load()
	if err != nil {
		return
	}
	if len(entries) > keep {
		entries = entries[len(entries)-keep:]
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".usage.*")
	if err != nil {
		return
	}
	w := bufio.NewWriter(tmp)
	for _, e := range entries {
		line, _ := json.Marshal(e)
		w.Write(append(line, '\n'))
	}
	if w.Flush() != nil || tmp.Close() != nil {
		os.Remove(tmp.Name())
		return
	}
	os.Chmod(tmp.Name(), 0o600)
	os.Rename(tmp.Name(), path)
}

// Load reads every entry, oldest first. No log is no entries.
func Load() ([]Entry, error) {
	f, err := os.Open(Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.ID != "" {
			out = append(out, e)
		}
	}
	return out, sc.Err()
}

// Clear removes the log.
func Clear() error {
	err := os.Remove(Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Stat is what the log says about one id.
type Stat struct {
	ID    string
	Kind  string
	Title string
	Count int
	Last  time.Time
	First time.Time
}

// Stats folds the entries into one Stat per id, most used first, ties by
// most recent.
func Stats(entries []Entry) []Stat {
	byID := map[string]*Stat{}
	var order []string
	for _, e := range entries {
		s, ok := byID[e.ID]
		if !ok {
			s = &Stat{ID: e.ID, Kind: e.Kind, Title: e.Title, First: e.Time}
			byID[e.ID] = s
			order = append(order, e.ID)
		}
		s.Count++
		if e.Time.After(s.Last) {
			s.Last = e.Time
		}
		if e.Title != "" {
			s.Title = e.Title
		}
	}
	out := make([]Stat, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Last.After(out[j].Last)
	})
	return out
}

// Recent is the n ids opened most recently, newest first.
func Recent(entries []Entry, n int) []string {
	seen := map[string]bool{}
	var ids []string
	for i := len(entries) - 1; i >= 0 && len(ids) < n; i-- {
		id := entries[i].ID
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// Front puts the items opened most recently first, up to n of them, each
// marked in its subtitle, and the rest after in their own order. An id
// in the log that is not in items (an app since removed, a row a typed
// query did not produce) is skipped. The marked items are not repeated
// below: they are at the top, which is where the eye looks first.
func Front(items []protocol.Item, recent []string, n int) []protocol.Item {
	if n <= 0 || len(recent) == 0 {
		return items
	}
	index := map[string]int{}
	for i, it := range items {
		index[it.ID] = i
	}
	var front []protocol.Item
	taken := map[int]bool{}
	for _, id := range recent {
		if i, ok := index[id]; ok && !taken[i] {
			it := items[i]
			it.Subtitle = mark(it.Subtitle)
			front = append(front, it)
			taken[i] = true
			if len(front) == n {
				break
			}
		}
	}
	if len(front) == 0 {
		return items
	}
	out := make([]protocol.Item, 0, len(items))
	out = append(out, front...)
	for i, it := range items {
		if !taken[i] {
			out = append(out, it)
		}
	}
	return out
}

// Mark is the word in a recent row's subtitle.
const Mark = "recent"

func mark(subtitle string) string {
	if strings.TrimSpace(subtitle) == "" {
		return "\x1b[2m" + Mark + "\x1b[22m"
	}
	return subtitle + "  \x1b[2m" + Mark + "\x1b[22m"
}
