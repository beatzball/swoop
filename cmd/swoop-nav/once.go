package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/tool"
)

// envOnce names the file that holds, for this run, the rows of the
// extensions that asked to be listed once per launch (ext.CacheFile),
// pictures included. Root reloads read it instead of listing them again:
// their rows do not change while the launcher is open, and a reload
// happens on every keystroke. bin/swoop names it beside the state file;
// the first `swoop-nav rows` of the run fills it.
const envOnce = "SWOOP_ONCE"

// envIcons names the file the pictures for those rows go to, which
// bin/swoop sends to the terminal once fzf has started. It is set for the
// first `swoop-nav rows` of a run only: the terminal takes them once.
const envIcons = "SWOOP_ICONS"

// rootRows is every extension's rows for the root: the kept rows of the
// ones listed once per launch, and what the rest answer for the text.
// The first call of a run lists both sorts side by side, so a long list
// kept for the run costs the start no more than the slowest extension.
//
// That first call, the one bin/swoop waits for before fzf starts, gives
// a list kept for the run the limit of the start and not a keystroke's
// (ext.ListStart): it is asked once, and the first paint without it has
// no applications. The rest keep a keystroke's limit there too. Any list
// that still misses its limit at the start is written to the late log,
// and leaves a file for bin/swoop, which then asks again as fzf starts,
// with no key pressed (see late.go).
//
// The kept rows of an extension turned off since are left out, and one
// turned on since is listed then, late: without pictures, which the
// terminal took at the start. One whose list failed is not kept, and is
// asked again at the next reload, as any other extension is.
func rootRows(query string) []protocol.Item {
	var once, live []ext.Extension
	for _, e := range ext.Discover(ext.Dirs()) {
		if e.Once() {
			once = append(once, e)
		} else {
			live = append(live, e)
		}
	}
	// With no path swoop-nav was run by hand: nowhere to keep the rows,
	// and no pictures.
	path := os.Getenv(envOnce)
	icons := os.Getenv(envIcons)
	start := icons != ""
	var names []string
	var kept []protocol.Item
	if path != "" && len(once) > 0 {
		names, kept = readOnce(path)
	}
	var late []ext.Extension
	for _, e := range once {
		if !slices.Contains(names, e.Name) {
			late = append(late, e)
		}
	}
	var fresh []protocol.Item
	var listed []string
	var errs []error
	var wg sync.WaitGroup
	if len(late) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fresh, listed, errs = ext.ListEach(late, func(e ext.Extension) ([]protocol.Item, error) {
				if start {
					return e.ListStart()
				}
				return e.List("")
			})
			if path != "" {
				fresh = pictures(fresh, late, icons)
			}
		}()
	}
	rest, _, liveErrs := ext.ListEach(live, func(e ext.Extension) ([]protocol.Item, error) { return e.List(query) })
	wg.Wait()
	failed(append(errs, liveErrs...), start, path)
	kept = append(kept, fresh...)
	if len(listed) > 0 && path != "" {
		names = append(names, listed...)
		if err := writeOnce(path, names, kept); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-nav:", err)
		}
	}
	items := make([]protocol.Item, 0, len(kept)+len(rest))
	for _, it := range kept {
		name, _, _ := ext.Route(it.ID)
		if slices.ContainsFunc(once, func(e ext.Extension) bool { return e.Name == name }) {
			items = append(items, it)
		}
	}
	return append(items, rest...)
}

// failed says what the lists that failed said: one line on stderr each,
// and no rows from that extension. At the start, the ones that missed
// their limit are also written to the late log, and the file named by
// lateMark is left beside the once file, at path, for bin/swoop.
func failed(errs []error, start bool, path string) {
	var late []string
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, tool.Name()+":", err)
		var le *ext.LateError
		if errors.As(err, &le) {
			late = append(late, le.Error())
		}
	}
	if !start || len(late) == 0 {
		return
	}
	if err := logLate(late); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
	}
	if path == "" {
		return
	}
	if err := os.WriteFile(path+lateMark, nil, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
	}
}

// onceHeader starts the first line of the file, which names the
// extensions whose rows it holds: one with no rows is still listed, and
// is not asked again. No row starts this way, as an id is never empty.
const onceHeader = "\tonce\t"

// readOnce reads the kept rows and the names of the extensions they are
// from. A file that is not there yet is no names and no rows.
func readOnce(path string) (names []string, items []protocol.Item) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if rest, ok := strings.CutPrefix(line, onceHeader); ok {
			names = strings.Fields(rest)
			continue
		}
		if it, err := protocol.Parse(line); err == nil {
			items = append(items, it)
		}
	}
	return names, items
}

// writeOnce writes the file, whole or not at all, through a rename: a
// reload may be reading it.
func writeOnce(path string, names []string, items []protocol.Item) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f, onceHeader+strings.Join(names, " "))
	if err == nil {
		err = protocol.Write(f, items)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

// pictures hands the rows of the extensions that asked for icons
// (ext.IconsFile) to swoop-icons, which puts picture cells in place of
// each glyph and writes the pictures themselves to out. With no out the
// pictures have nowhere to go, and swoop-icons is told so: it only pads
// each glyph to the two columns a picture takes. If swoop-icons fails,
// the rows keep their glyphs and the list still works.
func pictures(items []protocol.Item, exts []ext.Extension, out string) []protocol.Item {
	wants := map[string]bool{}
	for _, e := range exts {
		if e.IconsByID() {
			wants[e.Name] = true
		}
	}
	var in bytes.Buffer
	var at []int
	for i, it := range items {
		if name, _, _ := ext.Route(it.ID); wants[name] {
			at = append(at, i)
			in.WriteString(it.Line() + "\n")
		}
	}
	if len(at) == 0 {
		return items
	}
	cmd := exec.Command("swoop-icons", "-out", out)
	if out == "" {
		cmd = exec.Command("swoop-icons", "-out", os.DevNull)
		cmd.Env = append(os.Environ(), "SWOOP_PICTURES=0")
	}
	cmd.Stdin = &in
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-nav:", err)
		return items
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != len(at) {
		return items
	}
	for n, i := range at {
		if it, err := protocol.Parse(lines[n]); err == nil && it.ID == items[i].ID {
			items[i] = it
		}
	}
	return items
}
