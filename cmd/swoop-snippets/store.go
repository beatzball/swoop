// The snippets folder and what a snippet file is. One markdown file per
// snippet in ~/.config/<name>/snippets/: the first line is the name (a
// leading "# " is allowed, so the file reads as markdown), an optional
// `keyword:` line follows, and the rest is the text. Blank lines between
// the header and the text are not part of it, and neither is the one
// newline an editor leaves at the end.
//
//	Signature
//	keyword: ;sig
//
//	Best,
//	Sam
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/beatzball/swoop/internal/settings"
)

// Snippet is one file.
type Snippet struct {
	File    string // the file's name in the folder; also the row's id
	Name    string
	Keyword string
	Text    string // only read in full by read(); heads() leaves it empty
}

// ext is the files that are snippets. Anything else in the folder (a
// README of the user's own, a .git) is left alone.
const ext = ".md"

// dir is the folder, ~/.config/<name>/snippets.
func dir() string {
	d := settings.Dir()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "snippets")
}

// heads reads the name and keyword of every snippet, sorted by file
// name. It is the hot path: `list` runs on every keystroke at the root,
// so it reads the first two lines of a file and nothing more, and only
// when the file changed since the last list (see cache.go). A folder
// that is not there is no snippets, not an error.
func heads() ([]Snippet, error) {
	d := dir()
	entries, err := os.ReadDir(d)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return headsCached(d, entries), nil
}

func isSnippet(name string) bool {
	return strings.HasSuffix(name, ext) && !strings.HasPrefix(name, ".")
}

// head reads a file's first two lines. One read of 512 bytes holds them
// for nearly every snippet; a longer header is read on until the second
// newline. A file that cannot be read, or has no name, is skipped: one
// bad file must not hide the rest.
func head(path string) (Snippet, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Snippet{}, false
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 512)
	first, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return Snippet{}, false
	}
	second, _ := r.ReadString('\n')
	s, _ := parseHeader(first, second)
	return s, s.Name != ""
}

// parseHeader reads the name line and the line after it. keyword says
// whether the second line was the keyword, and so part of the header.
func parseHeader(first, second string) (s Snippet, keyword bool) {
	s.Name = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(first), "#"))
	k, ok := cutFold(strings.TrimSpace(second), "keyword:")
	if ok {
		s.Keyword = strings.TrimSpace(k)
	}
	return s, ok
}

// cutFold is strings.CutPrefix, ignoring case: Keyword: and keyword:
// are the same line.
func cutFold(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):], true
	}
	return s, false
}

// parse reads a whole file.
func parse(data string) Snippet {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	lines := strings.SplitAfter(data, "\n")
	second := ""
	if len(lines) > 1 {
		second = lines[1]
	}
	s, keyword := parseHeader(lines[0], second)
	rest := lines[1:]
	if keyword {
		rest = rest[1:]
	}
	for len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
		rest = rest[1:]
	}
	s.Text = strings.TrimSuffix(strings.Join(rest, ""), "\n")
	return s
}

// read loads one snippet in full, by its file name. A name with a path
// separator in it is not a file in the folder, and is refused.
func read(file string) (Snippet, error) {
	if file == "" || file != filepath.Base(file) || !isSnippet(file) {
		return Snippet{}, fmt.Errorf("no snippet %q", file)
	}
	data, err := os.ReadFile(filepath.Join(dir(), file))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snippet{}, fmt.Errorf("no snippet %q", file)
		}
		return Snippet{}, err
	}
	s := parse(string(data))
	s.File = file
	return s, nil
}

// format is the file for a snippet, the way parse reads it back.
func format(s Snippet) string {
	var b strings.Builder
	b.WriteString(oneLine(s.Name) + "\n")
	if k := oneLine(s.Keyword); k != "" {
		b.WriteString("keyword: " + k + "\n")
	}
	b.WriteString("\n" + s.Text + "\n")
	return b.String()
}

// oneLine keeps a header field on its line: a newline inside a name would
// make the rest of it the text.
func oneLine(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(s))
}

// put writes snippets into the folder. One with the name of a snippet
// already there replaces that file: adding is also how one is corrected.
// A new one gets a file named after it. Each file is written through a
// temporary file and a rename, so a crash leaves the old one whole.
func put(snippets []Snippet) error {
	d := dir()
	if d == "" {
		return errors.New("no config directory")
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	have, err := heads()
	if err != nil {
		return err
	}
	byName := map[string]string{}
	taken := map[string]bool{}
	for _, h := range have {
		byName[h.Name] = h.File
		taken[h.File] = true
	}
	for _, s := range snippets {
		s.Name = oneLine(s.Name)
		file, ok := byName[s.Name]
		if !ok {
			file = freeName(slug(s.Name), taken)
			taken[file] = true
			byName[s.Name] = file
		}
		p := filepath.Join(d, file)
		if err := os.WriteFile(p+".tmp", []byte(format(s)), 0o600); err != nil {
			return err
		}
		if err := os.Rename(p+".tmp", p); err != nil {
			return err
		}
	}
	return nil
}

// slug is a file name from a snippet's name: lower case, letters and
// digits, words joined by dashes. Letters outside ASCII stay, so a name
// in another script still names its file.
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r == '/' || r == '\\' || r == '.' || r < ' ' || strings.ContainsRune(` :*?"<>|'`+"`", r):
			dash = b.Len() > 0
		case r == ' ' || r == '-' || r == '_' || r == '\t':
			dash = b.Len() > 0
		default:
			if dash {
				b.WriteByte('-')
				dash = false
			}
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "snippet"
	}
	return b.String()
}

// freeName is base + ext, or base-2 + ext and on, whichever is not taken.
func freeName(base string, taken map[string]bool) string {
	name := base + ext
	for n := 2; taken[name]; n++ {
		name = fmt.Sprintf("%s-%d%s", base, n, ext)
	}
	return name
}

// remove deletes one snippet's file.
func remove(file string) error {
	if _, err := read(file); err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir(), file))
}
