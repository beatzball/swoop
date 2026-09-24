// Package clip is the clipboard history on disk: one JSON object per line,
// appended by the watcher, read by the extension. It knows nothing about
// pasteboards; it only stores text with a time, and hands it back newest
// first.
package clip

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Entry is one thing that was copied.
type Entry struct {
	// ID is the time it was seen, in nanoseconds. Unique enough, stable,
	// and sortable; it is what the extension gets back on Enter.
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

// MaxText is the most of one entry that is kept. A launcher row and a
// preview pane cannot use more, and a copied file's contents should not
// grow the history without bound.
const MaxText = 64 << 10

// MaxEntries is how many entries the file keeps after compaction; it is
// compacted when it grows past twice that, so a compaction is rare and
// the file never holds more than 2*MaxEntries lines.
const MaxEntries = 500

// Store is a history file.
type Store struct {
	Path string
}

// Default is the store in the user's data directory: ~/.local/share, or
// XDG_DATA_HOME when set, on every OS for now.
func Default() Store {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return Store{Path: filepath.Join(base, "swoop", "clipboard", "history.jsonl")}
}

// Append records text, unless it is empty or the same as the newest entry:
// copying the same thing twice is one thing. Text past MaxText is cut.
func (s Store) Append(text string, now time.Time) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if len(text) > MaxText {
		text = text[:MaxText]
	}
	if last, err := s.newest(); err == nil && last != nil && last.Text == text {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(Entry{ID: now.UnixNano(), Text: text})
	if err != nil {
		return err
	}
	// 0600: the history is the user's own business.
	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	// Closed before compacting, not deferred: compact renames a new file
	// over this one, and Windows refuses to replace a file that is open.
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	return s.compact()
}

// All returns every entry, newest first. A line that does not parse is
// skipped: one damaged line must not hide the history.
func (s Store) All() ([]Entry, error) {
	f, err := os.Open(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var entries []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 4*MaxText)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.ID != 0 {
			entries = append(entries, e)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// Find returns the entries whose text contains query, ignoring case,
// newest first. An empty query returns everything.
func (s Store) Find(query string) ([]Entry, error) {
	all, err := s.All()
	if err != nil || query == "" {
		return all, err
	}
	q := strings.ToLower(query)
	var out []Entry
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.Text), q) {
			out = append(out, e)
		}
	}
	return out, nil
}

// Get returns the entry with that id.
func (s Store) Get(id string) (Entry, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return Entry{}, errors.New("clip: bad id")
	}
	all, err := s.All()
	if err != nil {
		return Entry{}, err
	}
	for _, e := range all {
		if e.ID == n {
			return e, nil
		}
	}
	return Entry{}, errors.New("clip: no such entry")
}

func (s Store) newest() (*Entry, error) {
	all, err := s.All()
	if err != nil || len(all) == 0 {
		return nil, err
	}
	return &all[0], nil
}

// compact rewrites the file with only the newest MaxEntries once it has
// grown past twice that. Written to a temp name and renamed, so a reader
// never sees half a file.
func (s Store) compact() error {
	all, err := s.All()
	if err != nil || len(all) <= 2*MaxEntries {
		return err
	}
	keep := all[:MaxEntries]
	tmp := s.Path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for i := len(keep) - 1; i >= 0; i-- {
		line, err := json.Marshal(keep[i])
		if err != nil {
			f.Close()
			return err
		}
		w.Write(append(line, '\n'))
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// Title is the one-line form of an entry for a row: the first line that
// has anything on it, spaces collapsed, cut to max runes with an ellipsis.
func Title(text string, max int) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		r := []rune(line)
		if len(r) > max {
			return string(r[:max-1]) + "…"
		}
		return line
	}
	return ""
}
