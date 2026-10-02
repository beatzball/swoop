// Package ext finds and runs extensions. An extension is a directory that
// holds one executable of the same name; the executable answers three
// verbs, each one process: `list [query]` prints result lines, `preview
// <id>` prints the pane for one result, `run <id>` performs it. That is
// the whole contract, spelled out in the "Spec: the line protocol" issue.
//
// The launcher owns routing. Rows from an extension get their id prefixed
// with Prefix and the extension's name; Route strips it again before the
// id goes back to the extension. The extension never sees the prefix.
package ext

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
	"github.com/beatzball/swoop/internal/tool"
)

// Prefix starts every id that belongs to an extension: "ext/<name>/<id>".
const Prefix = "ext/"

// EnvDirs names extra directories to search, colon-separated. bin/swoop
// sets it to the extensions bundled in the repository.
const EnvDirs = "SWOOP_EXTENSIONS"

// EnvExtDir is set for the extension process to its own directory, so a
// script can find files that ship next to it.
const EnvExtDir = "SWOOP_EXT_DIR"

// listTimeout bounds one extension's `list`. The launcher waits for every
// source before fzf starts, so one stuck extension would otherwise stall
// the whole list. Two seconds is generous; a source near it should cache.
var listTimeout = 2 * time.Second

// startTimeout bounds the list of an extension whose rows are kept for
// the run (CacheFile), when the launcher asks for it as it starts. That
// list is not one of a keystroke's: it is the launcher's own first paint,
// asked once, and a run without it has no applications. A warm list is
// 10 ms. The first run of a program that is new on the machine is not:
// macOS checks each new program once, one at a time, a fifth to a third
// of a second each, and the launcher starts every extension together, so
// the last of them has been seen to wait 8 seconds right after a build
// with the machine busy. Ten seconds waits that out, and still ends a
// start that a stuck extension would hold for good.
var startTimeout = 10 * time.Second

// LateError is a call that was stopped because it took longer than its
// limit. It is an error of its own so a caller can tell an extension that
// was slow, and is worth asking again, from one that failed.
type LateError struct {
	Name  string // the extension
	Verb  string // list, view, send
	Limit time.Duration
}

func (e *LateError) Error() string {
	return fmt.Sprintf("%s: %s took longer than %s", e.Name, e.Verb, e.Limit)
}

// Extension is one discovered extension.
type Extension struct {
	Name string
	Dir  string
	Exe  string
}

// Dirs returns the directories searched, in order: the user's, then any in
// EnvDirs. The user's config home is ~/.config, or XDG_CONFIG_HOME when
// set, on every OS for now; the Windows frame can add its own convention.
func Dirs() []string {
	var dirs []string
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".config")
		}
	}
	if base != "" {
		dirs = append(dirs, filepath.Join(base, tool.Name(), "extensions"))
	}
	for _, d := range strings.Split(os.Getenv(EnvDirs), string(os.PathListSeparator)) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// Discover returns the extensions in dirs that are on: every one, less
// the names in the settings file's off list. One that is off has no rows,
// no view and no actions, because every caller finds extensions here.
func Discover(dirs []string) []Extension {
	return skip(DiscoverAll(dirs), settings.OffList())
}

// skip drops the extensions named in off, keeping the order.
func skip(exts []Extension, off []string) []Extension {
	if len(off) == 0 {
		return exts
	}
	kept := exts[:0:0]
	for _, e := range exts {
		if !slices.Contains(off, e.Name) {
			kept = append(kept, e)
		}
	}
	return kept
}

// DiscoverAll returns every extension in dirs, on or off, sorted by name:
// what the Settings pane lists to turn them on and off. A directory
// without an executable of its own name is not an extension and is
// skipped without comment: a README or a data directory next to real
// extensions is normal. The first directory to define a name wins.
func DiscoverAll(dirs []string) []Extension {
	seen := map[string]bool{}
	var exts []Extension
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || seen[e.Name()] {
				continue
			}
			exe := filepath.Join(dir, e.Name(), e.Name())
			if !executable(exe) {
				continue
			}
			seen[e.Name()] = true
			exts = append(exts, Extension{Name: e.Name(), Dir: filepath.Join(dir, e.Name()), Exe: exe})
		}
	}
	sort.Slice(exts, func(i, j int) bool { return exts[i].Name < exts[j].Name })
	return exts
}

func executable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		// Windows has no execute bit; a file of the right name is enough.
		return true
	}
	return st.Mode()&0o111 != 0
}

// Find returns the extension called name, if it is installed and on.
func Find(name string) (Extension, bool) {
	for _, e := range Discover(Dirs()) {
		if e.Name == name {
			return e, true
		}
	}
	return Extension{}, false
}

// KeywordFile names the file beside an extension's executable that holds
// its keyword: one word on the first line, "def" for Define. A file, not
// a line inside the script, because an extension may be a compiled
// program with no comments to read, and because the launcher reads it
// without running anything. It is read only when the root bar has a word
// and a space in it, so a keystroke without one never touches the disk.
const KeywordFile = "keyword"

// Keyword returns the extension's keyword, or "" without one.
func (e Extension) Keyword() string {
	data, err := os.ReadFile(filepath.Join(e.Dir, KeywordFile))
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// SplitKeyword splits the root bar's text into a keyword and the rest.
// The keyword is the first word, and only once a space follows it: "def
// ap" is "def" and "ap", "def " is "def" and "". The word alone is not a
// keyword yet, because it is also the start of an app's name: "calc" is
// on its way to Calculator, "notes" may be Notes itself. Text that starts
// with a space has no keyword.
func SplitKeyword(query string) (kw, rest string, ok bool) {
	kw, rest, found := strings.Cut(query, " ")
	if !found || kw == "" {
		return "", "", false
	}
	return kw, strings.TrimLeft(rest, " "), true
}

// ByKeyword returns the extension whose keyword is kw, ignoring case.
// When two claim the same keyword, the first by name wins.
func ByKeyword(exts []Extension, kw string) (Extension, bool) {
	for _, e := range exts {
		if k := e.Keyword(); k != "" && strings.EqualFold(k, kw) {
			return e, true
		}
	}
	return Extension{}, false
}

// Route splits an id of the form "ext/<name>/<id>" into its parts. ok is
// false for any other id, which is then nobody's: every row is an
// extension's.
func Route(id string) (name, rawID string, ok bool) {
	rest, found := strings.CutPrefix(id, Prefix)
	if !found {
		return "", "", false
	}
	name, rawID, found = strings.Cut(rest, "/")
	if !found || name == "" || rawID == "" {
		return "", "", false
	}
	return name, rawID, true
}

func (e Extension) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, e.Exe, args...)
	cmd.Dir = e.Dir
	cmd.Env = append(os.Environ(), EnvExtDir+"="+e.Dir)
	return cmd
}

// List runs `<exe> list [query]` and returns its rows with ids prefixed.
// A line that is not a protocol line is skipped, not fatal: one bad line
// should not hide the good ones. A non-zero exit is an error and hides
// all of them, because the output can no longer be trusted.
func (e Extension) List(query string) ([]protocol.Item, error) {
	args := []string{"list"}
	if query != "" {
		args = append(args, query)
	}
	return e.rows(listTimeout, args...)
}

// ListStart runs `<exe> list` with no text, as List does, with the limit
// of the launcher's start in place of a keystroke's: for the rows kept
// for the run (see Once), asked as the launcher starts. See startTimeout.
func (e Extension) ListStart() ([]protocol.Item, error) {
	return e.rows(startTimeout, "list")
}

// View runs `<exe> view <id> [query]`: the rows of the pane that opens
// when the user presses Enter on one of the extension's "view" rows. The
// extension does the filtering and chooses the order; the launcher shows
// exactly what comes back. Same prefixing and same rules as List.
func (e Extension) View(viewID, query string) ([]protocol.Item, error) {
	args := []string{"view", viewID}
	if query != "" {
		args = append(args, query)
	}
	return e.rows(listTimeout, args...)
}

func (e Extension) rows(limit time.Duration, args ...string) ([]protocol.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := e.command(ctx, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &LateError{Name: e.Name, Verb: args[0], Limit: limit}
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s: %s: %s", e.Name, args[0], msg)
	}
	var items []protocol.Item
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		it, err := protocol.Parse(sc.Text())
		if err != nil {
			continue
		}
		it.ID = Prefix + e.Name + "/" + it.ID
		items = append(items, it)
	}
	return items, nil
}

// ListEach runs list for every extension, all at once, and returns the
// rows of those that answered, their names, and the errors of the rest;
// the others still count. The order of the rows is by extension name,
// then the extension's own order. Nothing is printed: what to say about a
// failure, and whether to ask again, is the caller's.
func ListEach(exts []Extension, list func(Extension) ([]protocol.Item, error)) (items []protocol.Item, names []string, errs []error) {
	results := make([][]protocol.Item, len(exts))
	failed := make([]error, len(exts))
	var wg sync.WaitGroup
	for i, e := range exts {
		wg.Add(1)
		go func(i int, e Extension) {
			defer wg.Done()
			results[i], failed[i] = list(e)
		}(i, e)
	}
	wg.Wait()
	for i, e := range exts {
		if failed[i] != nil {
			errs = append(errs, failed[i])
			continue
		}
		names = append(names, e.Name)
		items = append(items, results[i]...)
	}
	return items, names, errs
}

// Run runs `<exe> run <id> [action]` with the terminal's own streams, so
// an action that wants to say something can. An empty action is the
// default one, Enter's.
func (e Extension) Run(rawID, action string) error {
	args := []string{"run", rawID}
	if action != "" {
		args = append(args, action)
	}
	cmd := e.command(context.Background(), args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// Actions runs `<exe> actions <id>`: the rows of the menu ctrl-k opens on
// one of the extension's rows. The ids are action names, not prefixed:
// they route through the pane's target, not on their own. An extension
// without the verb answers with an error or nothing, and either means "no
// actions" to the caller.
func (e Extension) Actions(rawID string) []protocol.Item {
	items, err := e.rows(listTimeout, "actions", rawID)
	if err != nil {
		return nil
	}
	for i := range items {
		items[i].ID = strings.TrimPrefix(items[i].ID, Prefix+e.Name+"/")
	}
	return items
}

// Preview runs `<exe> preview <id>` and copies its output to w.
func (e Extension) Preview(rawID string, w io.Writer) error {
	cmd := e.command(context.Background(), "preview", rawID)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
