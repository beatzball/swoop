// Package tool reads the one file that names the tool built on swoopkit:
// a file called tool, beside the extensions folder. A few words, one per
// line, the key and then the value:
//
//	name mytool
//	title My Tool
//	id com.example.mytool
//	hotkey alt+space
//
// Everything a person sees as the tool's own comes from it: the folders
// (~/.config/<name> and the rest), the launchd labels (<id>.shell and
// <id>.clipd), what the frame shows (<title>), and the key that opens it.
// The kit's own names do not: its programs and its variables keep the
// kit's prefix under every tool, so an extension written for one tool
// runs under another.
//
// The launcher script in bin/ reads the same file, in shell, for its
// `tool` verb; the test in this package holds the two to one answer.
package tool

import (
	"os"
	"path/filepath"
	"strings"
)

// Kit is the toolkit's own prefix, the one its programs and variables
// carry. It is also the name of a tool that has no file, which is how
// the kit run bare, and every install older than the file, keep the
// folders they had.
const Kit = "swoop"

// Env names the file. The launcher sets it for everything it runs, so
// every program in one run reads one file; a program run by hand, with
// the variable unset, looks beside its own bin/ instead.
const Env = "SWOOP_TOOL"

// File is the file's name.
const File = "tool"

// hotkey is the key that opens the frame of a tool whose file names none.
const hotkey = "alt+shift+space"

// maxSize is more than any real file holds. A file past it is something
// else that happens to be called tool, and is not read.
const maxSize = 4096

// Tool is the file, with what it leaves out filled in.
type Tool struct {
	// Name is the word in every path: ~/.config/<name>, the log folder,
	// the command a person types.
	Name string
	// Title is the name as shown: the frame's menu, a notification.
	Title string
	// ID is the reverse-DNS prefix of the launchd labels and of the
	// frame's signature: <id>.shell, <id>.clipd.
	ID string
	// Hotkey opens the frame until the user picks a key of their own.
	Hotkey string
}

// Path is where the file is looked for: the one Env names, or tool beside
// the bin/ this program runs from. Links are followed first: a package
// manager links the programs into its own bin/, and the file is beside
// the real one.
func Path() string {
	if p := os.Getenv(Env); p != "" {
		return p
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Join(filepath.Dir(exe), "..", File)
}

// Read is the tool this program runs as. No file, or a file that cannot
// be read, is the kit under its own name: a missing file is the common
// case, not an error.
func Read() Tool {
	path := Path()
	if path == "" {
		return Parse("")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxSize {
		return Parse("")
	}
	return Parse(string(data))
}

// Name is Read().Name, for the many callers that want only the folder.
func Name() string { return Read().Name }

// Parse reads the file's text. A line is a key, a space, and the value;
// a blank line and a line that starts with # are skipped, and so is a
// key this version does not know, so a newer file still reads. A name or
// an id that is not one plain word is skipped too: both end up in paths
// and labels. What is left out is derived: the title is the name, the id
// is dev.<name>.
func Parse(text string) Tool {
	var t Tool
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value := line, ""
		if i := strings.IndexAny(line, " \t"); i >= 0 {
			key, value = line[:i], strings.TrimSpace(line[i:])
		}
		switch key {
		case "name":
			if Word(value) {
				t.Name = value
			}
		case "title":
			t.Title = value
		case "id":
			if Word(value) {
				t.ID = value
			}
		case "hotkey":
			if Word(value) {
				t.Hotkey = value
			}
		}
	}
	if t.Name == "" {
		t.Name = Kit
	}
	if t.Title == "" {
		t.Title = t.Name
	}
	if t.ID == "" {
		t.ID = "dev." + t.Name
	}
	if t.Hotkey == "" {
		t.Hotkey = hotkey
	}
	return t
}

// Word reports whether s can stand in a path and in a launchd label:
// letters, digits, and . _ - + after the first character, nothing else.
func Word(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case i > 0 && (r == '.' || r == '_' || r == '-' || r == '+'):
		default:
			return false
		}
	}
	return true
}
