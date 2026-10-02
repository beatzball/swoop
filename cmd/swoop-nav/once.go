package main

import (
	"bufio"
	"bytes"
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
	if len(once) == 0 {
		return ext.ListAll(live, query)
	}
	path := os.Getenv(envOnce)
	if path == "" {
		// swoop-nav run by hand: nowhere to keep them, no pictures.
		return append(ext.ListAll(once, ""), ext.ListAll(live, query)...)
	}
	names, kept := readOnce(path)
	var late []ext.Extension
	for _, e := range once {
		if !slices.Contains(names, e.Name) {
			late = append(late, e)
		}
	}
	var fresh []protocol.Item
	var listed []string
	var wg sync.WaitGroup
	if len(late) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fresh, listed = listOnce(late)
			fresh = pictures(fresh, late, os.Getenv(envIcons))
		}()
	}
	rest := ext.ListAll(live, query)
	wg.Wait()
	if len(listed) > 0 {
		names = append(names, listed...)
		kept = append(kept, fresh...)
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

// listOnce lists each extension with no text, all at once, and returns
// their rows, in the extensions' order, and the names of those that
// answered. A failure is one line on stderr and no rows, as in
// ext.ListAll, and its name is left out.
func listOnce(exts []ext.Extension) (items []protocol.Item, names []string) {
	results := make([][]protocol.Item, len(exts))
	failed := make([]bool, len(exts))
	var wg sync.WaitGroup
	for i, e := range exts {
		wg.Add(1)
		go func(i int, e ext.Extension) {
			defer wg.Done()
			rows, err := e.List("")
			if err != nil {
				fmt.Fprintln(os.Stderr, tool.Name()+":", err)
				failed[i] = true
				return
			}
			results[i] = rows
		}(i, e)
	}
	wg.Wait()
	for i, e := range exts {
		if !failed[i] {
			names = append(names, e.Name)
			items = append(items, results[i]...)
		}
	}
	return items, names
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
