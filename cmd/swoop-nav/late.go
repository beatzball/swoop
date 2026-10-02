package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/usage"
)

// The late log. A list that misses its limit as the launcher starts
// leaves the first paint short of its rows, and nothing on the screen
// says why: the line on stderr is under fzf a moment later. So each one
// is a line in late.log, beside the usage log: the time, a tab, and what
// was late, "apps: list took longer than 10s". `<name> status` prints
// the newest, through `swoop-nav late`.

// lateMark ends the name of the file the first `swoop-nav rows` of a run
// leaves beside the once file when a list missed its limit. bin/swoop
// looks for it once, before fzf starts, and when it is there has fzf ask
// for the rows again as it starts, with no key pressed: by then the slow
// start is over, and the rows that were missing fill themselves in.
const lateMark = ".late"

// lateKeep is the size the log is cut back from: half of it goes, the
// oldest half. A miss is rare, and a line is under a hundred bytes.
const lateKeep = 64 << 10

// latePath is the log: late.log in the folder of the usage log, or ""
// with no home to put it in.
func latePath() string {
	p := usage.Path()
	if p == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(p), "late.log")
}

// logLate appends one line for each of what, all with the time now.
func logLate(what []string) error {
	path := latePath()
	if path == "" || len(what) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if st, err := os.Stat(path); err == nil && st.Size() > lateKeep {
		if data, err := os.ReadFile(path); err == nil {
			data = data[len(data)/2:]
			if _, rest, ok := bytes.Cut(data, []byte("\n")); ok {
				data = rest
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return err
			}
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	var lines strings.Builder
	for _, w := range what {
		lines.WriteString(now + "\t" + w + "\n")
	}
	_, err = f.WriteString(lines.String())
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// lastLate is what the newest start with a late list logged: one line
// for each list, as `<name> status` shows them. Nothing when no start
// ever had one.
func lastLate() []string {
	data, err := os.ReadFile(latePath())
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	newest, _, _ := strings.Cut(lines[len(lines)-1], "\t")
	var out []string
	for _, line := range lines {
		at, what, ok := strings.Cut(line, "\t")
		if !ok || at != newest {
			continue
		}
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			at = t.Format("2006-01-02 15:04:05")
		}
		out = append(out, fmt.Sprintf("late:    %s as the launcher started, %s", what, at))
	}
	return out
}
