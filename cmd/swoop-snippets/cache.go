// The heads cache. `list` runs on every keystroke at the root, and
// opening a few hundred files to read two lines each took 7 ms; a stat
// of each is under 1 ms. So the names and keywords are kept in one file
// under the user's cache directory, each with the size and time of the
// file it came from, and a file is opened again only when either
// changed. A stat per file, and not the folder's time alone, because an
// editor that saves in place leaves the folder's time as it was.
//
// One line per snippet, tab-separated: file, size, time in nanoseconds,
// name, keyword. The first line names the folder, so a cache made for
// another one (a test's, another XDG_CONFIG_HOME) is not used for this.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const cacheVersion = "swoop-snippets 1"

// cached is one line of the cache.
type cached struct {
	size, mtime int64
	Snippet
}

// cachePath is ~/Library/Caches/swoop/snippets.tsv on a Mac,
// ~/.cache/swoop/snippets.tsv on Linux, or "" for no cache.
func cachePath() string {
	d, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "swoop", "snippets.tsv")
}

// loadCache reads the cache for folder. A cache that is missing, for
// another folder, or of another version is empty.
func loadCache(folder string) map[string]cached {
	out := map[string]cached{}
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return out
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, 1<<20)
	if !sc.Scan() || sc.Text() != cacheVersion+"\t"+folder {
		return out
	}
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 5 {
			continue
		}
		size, err1 := strconv.ParseInt(f[1], 10, 64)
		mtime, err2 := strconv.ParseInt(f[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out[f[0]] = cached{size, mtime, Snippet{File: f[0], Name: f[3], Keyword: f[4]}}
	}
	return out
}

// saveCache writes the cache through a temporary file and a rename. It
// is only a cache: a failure is ignored, and the next list reads the
// files again.
func saveCache(folder string, entries []cached) {
	p := cachePath()
	if p == "" || os.MkdirAll(filepath.Dir(p), 0o700) != nil {
		return
	}
	var b strings.Builder
	b.WriteString(cacheVersion + "\t" + folder + "\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%s\t%d\t%d\t%s\t%s\n", e.File, e.size, e.mtime, noTab(e.Name), noTab(e.Keyword))
	}
	tmp := p + ".tmp" + strconv.Itoa(os.Getpid())
	if os.WriteFile(tmp, []byte(b.String()), 0o600) != nil {
		return
	}
	if os.Rename(tmp, p) != nil {
		_ = os.Remove(tmp)
	}
}

func noTab(s string) string { return strings.ReplaceAll(s, "\t", " ") }

// headsCached is heads through the cache: entries are the folder's
// files, read with fs.ReadDir.
func headsCached(folder string, entries []fs.DirEntry) []Snippet {
	have := loadCache(folder)
	var out []Snippet
	var keep []cached
	changed := false
	for _, e := range entries {
		if e.IsDir() || !isSnippet(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		c, ok := have[e.Name()]
		if !ok || c.size != info.Size() || c.mtime != info.ModTime().UnixNano() {
			changed = true
			// A file with no name is kept too, nameless, so it is not
			// read again, and the cache written again, on every list.
			s, _ := head(filepath.Join(folder, e.Name()))
			s.File = e.Name()
			c = cached{info.Size(), info.ModTime().UnixNano(), s}
		}
		keep = append(keep, c)
		if c.Name != "" {
			out = append(out, c.Snippet)
		}
	}
	if changed || len(keep) != len(have) {
		saveCache(folder, keep)
	}
	return out
}
