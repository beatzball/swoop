// The notes folder and what a note is. One markdown file per note in
// ~/.local/share/swoop/notes/ (XDG_DATA_HOME when set): the first line
// is the title, a leading "# " allowed, so a note starts with a heading
// and reads as markdown anywhere. Files are the format, so any editor is
// the editor and any sync is the sync. A deleted note moves to deleted/
// in the folder, which is not listed, so a slip can be undone by hand.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Note is one file.
type Note struct {
	File  string // the file's name in the folder; also the row's id
	Title string
	Mod   time.Time
	Line  string // for a note found by its content, the line that matched
}

// ext is the files that are notes. Anything else in the folder is left
// alone.
const ext = ".md"

// deletedDir is where Delete moves a note, inside the folder.
const deletedDir = "deleted"

// maxRead caps how much of one file a search reads. A note is a few
// kilobytes; a pasted log of a hundred megabytes must not stall every
// keystroke in the view.
const maxRead = 256 << 10

// maxRows caps the rows a view prints. fzf is not filtering in a view,
// so every row is a reload on each keystroke.
const maxRows = 200

// dir is the folder.
func dir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "swoop", "notes")
}

func isNote(name string) bool {
	return strings.HasSuffix(name, ext) && !strings.HasPrefix(name, ".")
}

// Title is a note's title: its first line, without the leading "#"s of a
// heading. Only the first line, so listing a few hundred notes reads a
// few hundred bytes of each. A note whose first line is blank is called
// by its file name, so it still has a row that says which it is.
func Title(first, file string) string {
	t := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(first), "#"))
	if t == "" {
		return strings.TrimSuffix(file, ext)
	}
	return t
}

// firstLine is the text up to the first newline.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSuffix(line, "\r")
}

// scan lists the notes, the most recently changed first. Only the names
// and times: nothing is opened. A folder that is not there is no notes,
// not an error.
func scan(d string) ([]Note, error) {
	entries, err := os.ReadDir(d)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !isNote(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		notes = append(notes, Note{File: e.Name(), Mod: info.ModTime()})
	}
	Order(notes)
	return notes, nil
}

// Order sorts notes newest change first. Two changed in the same instant
// go by file name, so the order does not flicker between keystrokes.
func Order(notes []Note) {
	sort.Slice(notes, func(i, j int) bool {
		if !notes[i].Mod.Equal(notes[j].Mod) {
			return notes[i].Mod.After(notes[j].Mod)
		}
		return notes[i].File < notes[j].File
	})
}

// readHead reads a file's first line. One read of 512 bytes holds it for
// nearly every note.
func readHead(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	line, err := bufio.NewReaderSize(f, 512).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// readCapped reads a file up to maxRead bytes.
func readCapped(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRead))
	return string(data), err
}

// Match says whether a note holds every word of the query, each in the
// title or anywhere in the text, ignoring case. inTitle is true when the
// title alone holds them all; otherwise line is the first line of the
// text that holds the first word the title does not, to show why it
// matched. words are lower case.
func Match(title, text string, words []string) (ok, inTitle bool, line string) {
	lt := strings.ToLower(title)
	var body string
	inTitle = true
	for _, w := range words {
		if strings.Contains(lt, w) {
			continue
		}
		if body == "" {
			body = strings.ToLower(text)
		}
		i := strings.Index(body, w)
		if i < 0 {
			return false, false, ""
		}
		if inTitle {
			inTitle = false
			line = lineAt(text, body, i)
		}
	}
	return true, inTitle, line
}

// lineAt is the line of text around byte i of its lower-cased copy.
// Lower-casing can change a string's length, and then the offset is only
// near; the line is for show, so near is enough, kept on a character.
func lineAt(text, lower string, i int) string {
	if len(lower) != len(text) {
		text = lower
	}
	start := strings.LastIndexByte(text[:i], '\n') + 1
	end := strings.IndexByte(text[i:], '\n')
	if end < 0 {
		end = len(text)
	} else {
		end += i
	}
	s := strings.TrimSpace(text[start:end])
	for !utf8.ValidString(s) && s != "" {
		s = s[1:]
	}
	return s
}

// find is the view's rows: every note with no query; with one, the notes
// whose title holds it first, then those whose text does, each newest
// first. Capped at maxRows.
func find(d, query string) ([]Note, error) {
	notes, err := scan(d)
	if err != nil {
		return nil, err
	}
	words := strings.Fields(strings.ToLower(query))
	var titled, bodied []Note
	for _, n := range notes {
		if len(titled) >= maxRows {
			break
		}
		p := filepath.Join(d, n.File)
		if len(words) == 0 {
			first, err := readHead(p)
			if err != nil {
				continue
			}
			n.Title = Title(first, n.File)
			titled = append(titled, n)
			continue
		}
		text, err := readCapped(p)
		if err != nil {
			continue
		}
		n.Title = Title(firstLine(text), n.File)
		ok, inTitle, line := Match(n.Title, text, words)
		switch {
		case !ok:
		case inTitle:
			titled = append(titled, n)
		default:
			n.Line = line
			bodied = append(bodied, n)
		}
	}
	out := append(titled, bodied...)
	if len(out) > maxRows {
		out = out[:maxRows]
	}
	return out, nil
}

// path is a note's file, by its name. A name with a path separator in it
// is not a file in the folder, and is refused.
func path(file string) (string, error) {
	if file == "" || file != filepath.Base(file) || !isNote(file) {
		return "", fmt.Errorf("no note %q", file)
	}
	p := filepath.Join(dir(), file)
	if _, err := os.Stat(p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no note %q", file)
		}
		return "", err
	}
	return p, nil
}

// create writes a new note whose first line is title, as a heading, and
// returns its path. The file is named after the title and never replaces
// one already there.
func create(d, title string) (string, error) {
	if d == "" {
		return "", errors.New("no data directory")
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	title = strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(title))
	body := "\n"
	if title != "" {
		body = "# " + title + "\n\n"
	}
	base := slug(title)
	for n := 1; ; n++ {
		name := base + ext
		if n > 1 {
			name = fmt.Sprintf("%s-%d%s", base, n, ext)
		}
		p := filepath.Join(d, name)
		// O_EXCL: a note of the same name, even one made a moment ago
		// by another launcher, is never written over.
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = f.WriteString(body)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return p, err
	}
}

// slug is a file name from a title: lower case, letters and digits,
// words joined by dashes, at most 60 characters. Letters outside ASCII
// stay, so a title in another script still names its file.
func slug(title string) string {
	var b strings.Builder
	dash := false
	n := 0
	for _, r := range strings.ToLower(title) {
		if n >= 60 {
			break
		}
		switch {
		case r == '/' || r == '\\' || r == '.' || r < ' ' || strings.ContainsRune(` :*?"<>|'`+"`#-_\t", r):
			dash = b.Len() > 0
		default:
			if dash {
				b.WriteByte('-')
				dash = false
				n++
			}
			b.WriteRune(r)
			n++
		}
	}
	if b.Len() == 0 {
		return "untitled"
	}
	return b.String()
}

// remove moves a note into deleted/, under a free name there, and
// returns where it went.
func remove(d, file string) (string, error) {
	if file == "" || file != filepath.Base(file) || !isNote(file) {
		return "", fmt.Errorf("no note %q", file)
	}
	to := filepath.Join(d, deletedDir)
	if err := os.MkdirAll(to, 0o700); err != nil {
		return "", err
	}
	base := strings.TrimSuffix(file, ext)
	for n := 1; ; n++ {
		name := file
		if n > 1 {
			name = fmt.Sprintf("%s-%d%s", base, n, ext)
		}
		dst := filepath.Join(to, name)
		if _, err := os.Lstat(dst); err == nil {
			continue
		}
		if err := os.Rename(filepath.Join(d, file), dst); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("no note %q", file)
			}
			return "", err
		}
		return dst, nil
	}
}
