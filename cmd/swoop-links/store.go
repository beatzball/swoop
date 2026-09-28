// The quicklinks file and what a link is. One file, one link per line,
// three tab-separated fields: name, link, app to open it with. The third
// is optional. Blank lines and lines that start with # are skipped, so
// the file can carry a header for whoever edits it by hand.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/beatzball/swoop/internal/settings"
)

// Link is one quicklink.
type Link struct {
	Name string // shown as the row's title; also the row's id
	Link string // a URL, a path, or a deeplink; may hold {argument}
	App  string // what opens it, "" for the system default
}

// argument is the placeholder that makes a link a pane: it is replaced
// with what the user typed. Raycast's form is `{argument}`, with options
// inside the braces in newer exports; anything inside the braces after
// the word is accepted and ignored, so an export still imports.
var argument = regexp.MustCompile(`\{argument[^}]*\}`)

// HasArgument says whether Enter on the link asks for text first.
func (l Link) HasArgument() bool { return argument.MatchString(l.Link) }

// Fill puts arg in the link's placeholder. For a web link the text is
// query-encoded, because that is where it lands: "hello world" becomes
// hello+world, and an ampersand cannot cut the query short. Anything else
// (a file path, a deeplink) gets the text as typed.
func (l Link) Fill(arg string) string {
	if l.isWeb() {
		arg = url.QueryEscape(arg)
	}
	return argument.ReplaceAllLiteralString(l.Link, arg)
}

func (l Link) isWeb() bool {
	lower := strings.ToLower(l.Link)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// Where is the short form for a subtitle: the host of a web link, the
// link itself otherwise.
func (l Link) Where() string {
	if l.isWeb() {
		if u, err := url.Parse(argument.ReplaceAllLiteralString(l.Link, "x")); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return l.Link
}

// Icon is the row's glyph, by what the link points at: the web, a
// folder, a file, or something else (a deeplink another app owns).
func (l Link) Icon() string {
	switch {
	case l.isWeb():
		return "󰖟"
	case strings.HasPrefix(l.Link, "/") || strings.HasPrefix(l.Link, "~") || strings.HasPrefix(l.Link, "file://"):
		path := strings.TrimPrefix(l.Link, "file://")
		if strings.HasPrefix(path, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				path = home + path[1:]
			}
		}
		if st, err := os.Stat(path); err == nil && st.IsDir() {
			return ""
		}
		return ""
	}
	return ""
}

// defaults ship with swoop and are what the pane shows until the user's
// file exists. add and import start the user's file from them, so
// adding one link does not lose the five.
const defaults = `# swoop quicklinks: name, link, app to open it with; tab-separated.
# {argument} in a link asks for text first. Lines starting with # are ignored.
Google	https://www.google.com/search?q={argument}
DuckDuckGo	https://duckduckgo.com/?q={argument}
Wikipedia	https://en.wikipedia.org/w/index.php?search={argument}
YouTube	https://www.youtube.com/results?search_query={argument}
GitHub	https://github.com/search?q={argument}
`

// path is the user's file, ~/.config/swoop/quicklinks.tsv.
func path() string {
	dir := settings.Dir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "quicklinks.tsv")
}

// load reads the user's links, or the defaults when there is no file.
func load() ([]Link, error) {
	data, err := os.ReadFile(path())
	if errors.Is(err, os.ErrNotExist) {
		return parse([]byte(defaults)), nil
	}
	if err != nil {
		return nil, err
	}
	return parse(data), nil
}

// parse reads the file's lines. A line with fewer than two fields is not
// a link and is skipped, like a comment: one bad line must not hide the
// rest. Spaces around a field are trimmed, so a hand-aligned file reads
// the same as a tight one.
func parse(data []byte) []Link {
	var links []Link
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		f := strings.Split(line, "\t")
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		if len(f) < 2 || f[0] == "" || f[1] == "" {
			continue
		}
		l := Link{Name: f[0], Link: f[1]}
		if len(f) > 2 {
			l.App = f[2]
		}
		links = append(links, l)
	}
	return links
}

// find returns the link called name.
func find(links []Link, name string) (Link, bool) {
	for _, l := range links {
		if l.Name == name {
			return l, true
		}
	}
	return Link{}, false
}

// save writes the user's file, the whole thing, through a temporary file
// and a rename so a crash mid-write leaves the old file whole. The header
// comes back each time so the file stays self-describing.
func save(links []Link) error {
	p := path()
	if p == "" {
		return errors.New("no config directory")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# swoop quicklinks: name, link, app to open it with; tab-separated.\n")
	b.WriteString("# {argument} in a link asks for text first. Lines starting with # are ignored.\n")
	for _, l := range links {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", clean(l.Name), clean(l.Link), clean(l.App))
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// clean keeps a field on its line: a tab or a newline inside a name would
// split it into two fields on the next read.
func clean(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(strings.TrimSpace(s))
}
