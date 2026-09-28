package apps

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/beatzball/swoop/internal/protocol"
)

// This file reads freedesktop.org .desktop files, the way Linux desktops
// describe their applications. It has no build tag on purpose: the parser
// and the directory walk are plain Go, so their tests run on a Mac too.
// Only the choice of directories is Linux's, in apps_linux.go.

// Desktop is the part of a .desktop file's [Desktop Entry] group that swoop
// uses. Localised keys such as Name[de] are ignored: the plain key is the
// one every file has.
type Desktop struct {
	Type      string
	Name      string
	Exec      string
	Icon      string
	Comment   string
	NoDisplay bool
	Hidden    bool
}

// Listed says whether the entry belongs in the launcher. NoDisplay is an
// app that exists but asks not to be shown (a MIME handler, a settings
// panel); Hidden is an app the user or the system has deleted, usually by
// shadowing a system file with one in ~/.local/share/applications.
// OnlyShowIn and NotShowIn are ignored: swoop is not any one desktop.
func (d Desktop) Listed() bool {
	return d.Type == "Application" && d.Name != "" && !d.NoDisplay && !d.Hidden
}

// ParseDesktop reads a .desktop file. Only the [Desktop Entry] group
// counts; the [Desktop Action ...] groups after it repeat keys like Name
// and Exec for other purposes and must not overwrite the entry's own.
func ParseDesktop(r io.Reader) (Desktop, error) {
	var d Desktop
	in := false
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			in = line == "[Desktop Entry]"
			continue
		}
		if !in {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		// The spec allows spaces around the "=". A key with a [locale]
		// suffix never matches below, which is how those are skipped.
		key = strings.TrimSpace(key)
		value = unescape(strings.TrimSpace(value))
		switch key {
		case "Type":
			d.Type = value
		case "Name":
			d.Name = value
		case "Exec":
			d.Exec = value
		case "Icon":
			d.Icon = value
		case "Comment":
			d.Comment = value
		case "NoDisplay":
			d.NoDisplay = value == "true"
		case "Hidden":
			d.Hidden = value == "true"
		}
	}
	return d, sc.Err()
}

// ReadDesktop parses the .desktop file at path.
func ReadDesktop(path string) (Desktop, error) {
	f, err := os.Open(path)
	if err != nil {
		return Desktop{}, err
	}
	defer f.Close()
	return ParseDesktop(f)
}

// unescape applies the spec's escapes for string values: \s \n \t \r \\.
// Exec has a second layer of quoting on top of this, which sh understands,
// so it is left for sh to read.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 's':
			b.WriteByte(' ')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '\\':
			b.WriteByte('\\')
		default:
			// Not a string escape; keep both bytes, since Exec uses
			// backslashes of its own inside quoted arguments.
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// fieldCodes are the Exec placeholders a launcher fills in with files,
// URLs, the icon, the name, or the file's own path. swoop opens an app
// with nothing, so each is dropped. The deprecated ones (%d %D %n %N %v
// %m) are dropped too, as the spec says to.
var fieldCodes = "fFuUickdDnNvm"

// StripFieldCodes removes the field codes from an Exec value, leaving a
// command line sh can run. "%%" is a literal "%". The spaces around a
// dropped code stay; sh does not mind them, and collapsing them would
// also change spaces inside a quoted argument.
func StripFieldCodes(exec string) string {
	var b strings.Builder
	for i := 0; i < len(exec); i++ {
		if exec[i] != '%' || i+1 == len(exec) {
			b.WriteByte(exec[i])
			continue
		}
		i++
		switch {
		case exec[i] == '%':
			b.WriteByte('%')
		case strings.IndexByte(fieldCodes, exec[i]) >= 0:
			// dropped
		default:
			// An unknown code is invalid in the spec; keep it as written
			// rather than guess.
			b.WriteByte('%')
			b.WriteByte(exec[i])
		}
	}
	return strings.TrimSpace(b.String())
}

// desktopItems lists the applications in dirs, one level deep each, in the
// order given. A file name seen once is not looked at again: an earlier
// directory is a higher XDG precedence, so ~/.local/share/applications
// overrides /usr/share/applications file by file. That holds even when the
// winner is hidden, since a Hidden=true copy in the home directory is how
// a user removes a system app.
func desktopItems(dirs []string) []protocol.Item {
	seen := map[string]bool{}
	var items []protocol.Item
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// A missing directory is normal: most XDG_DATA_DIRS entries
			// have no applications folder. One unreadable directory is not
			// worth failing the whole list for.
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".desktop") || e.IsDir() || seen[name] {
				continue
			}
			seen[name] = true
			path := filepath.Join(dir, name)
			d, err := ReadDesktop(path)
			if err != nil || !d.Listed() {
				continue
			}
			items = append(items, protocol.Item{
				// The desktop file's path is the id: unique, stable, and
				// what `gio launch` takes. swoop-run gets it back as is.
				ID:    path,
				Kind:  Kind,
				Icon:  Icon,
				Title: d.Name,
				// No subtitle, as on the Mac: the path is in the preview.
			})
		}
	}
	return items
}
