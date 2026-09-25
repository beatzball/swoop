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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
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
const listTimeout = 2 * time.Second

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
		dirs = append(dirs, filepath.Join(base, "swoop", "extensions"))
	}
	for _, d := range strings.Split(os.Getenv(EnvDirs), string(os.PathListSeparator)) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// Discover returns every extension in dirs, sorted by name. A directory
// without an executable of its own name is not an extension and is
// skipped without comment: a README or a data directory next to real
// extensions is normal. The first directory to define a name wins.
func Discover(dirs []string) []Extension {
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

// Find returns the extension called name, if it is installed.
func Find(name string) (Extension, bool) {
	for _, e := range Discover(Dirs()) {
		if e.Name == name {
			return e, true
		}
	}
	return Extension{}, false
}

// Route splits an id of the form "ext/<name>/<id>" into its parts. ok is
// false for any other id, which then belongs to a built-in source.
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
	return e.rows(args...)
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
	return e.rows(args...)
}

func (e Extension) rows(args ...string) ([]protocol.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	cmd := e.command(ctx, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%s: %s took longer than %s", e.Name, args[0], listTimeout)
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

// ListAll runs every extension's list at once and returns all their rows.
// Each failure is one line on stderr and no rows from that extension; the
// others still count. The order of the result is by extension name, then
// the extension's own order.
func ListAll(exts []Extension, query string) []protocol.Item {
	results := make([][]protocol.Item, len(exts))
	var wg sync.WaitGroup
	for i, e := range exts {
		wg.Add(1)
		go func(i int, e Extension) {
			defer wg.Done()
			items, err := e.List(query)
			if err != nil {
				fmt.Fprintln(os.Stderr, "swoop:", err)
				return
			}
			results[i] = items
		}(i, e)
	}
	wg.Wait()
	var all []protocol.Item
	for _, r := range results {
		all = append(all, r...)
	}
	return all
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
	items, err := e.rows("actions", rawID)
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
