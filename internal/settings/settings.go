// Package settings reads and writes ~/.config/swoop/config, the one file
// for the launcher's own choices: what the user changed with a key and
// wants to find the same way next time. One `key = value` per line,
// `#` starts a comment, unknown keys are kept as they are. The frame's
// own file, shell-mac.json, is separate: it belongs to one OS.
package settings

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Path is the file: $XDG_CONFIG_HOME/swoop/config, or ~/.config/swoop/config.
func Path() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "swoop", "config")
}

// Preview is the key for the preview window's width, in percent of the
// whole, and PreviewDefault, PreviewMin and PreviewMax its range.
const (
	Preview        = "preview"
	PreviewDefault = 58
	PreviewMin     = 20
	PreviewMax     = 80
)

// Render is the key for the command that renders an AI answer, markdown
// on its stdin, styled text on its stdout: `render = glow -s dark`, or
// `render = swoop-md`. Unset, the built-in renderer runs in-process.
const Render = "render"

// Hotkey is the key that opens the frame, "mod+mod+key"; the frame reads
// it at start and whenever the file changes. SWOOP_HOTKEY in the frame's
// environment overrides it.
const (
	Hotkey        = "hotkey"
	HotkeyDefault = "alt+shift+space"
)

// AI names what answers in the Ask AI pane: a preset ("claude"), ollama
// with a model ("ollama:llama3.2"), or a command line of your own. Unset,
// the line in ~/.config/swoop/ai, then the defaults.
const AI = "ai"

// AIURL and AIKey are for the API kind of model, openai:<model> and
// lmstudio:<model>: the base URL ending in /v1, and the key. The file
// is mode 600, the user's alone, which is where a key belongs on a
// machine where launchd's environment is not yours to set.
const (
	AIURL = "ai_url"
	AIKey = "ai_key"
)

// Web says whether the model may search and fetch the web: "on" or "off".
const (
	Web        = "web"
	WebDefault = "off"
)

// Recent says whether the root list leads with what was opened most
// recently: "on" or "off".
const (
	Recent        = "recent"
	RecentDefault = "on"
)

// Skin is the emoji skin tone Enter uses: 0 for none (the yellow of
// the emoji as drawn), 1 to 5 from light to dark.
const (
	Skin        = "skin"
	SkinDefault = "0"
)

// EditorKey is the key for the program a note, the task list or a text
// file opens in, inside the panel: `editor = nvim`, `editor = emacs -nw`.
// Empty means $EDITOR, and with neither, EditorDefault, which every Mac
// and most Linux systems have. The name leaves Editor for the function
// that answers what to run, which is what every caller wants.
const (
	EditorKey     = "editor"
	EditorDefault = "nano"
)

// Off is the key for the extensions turned off, by name, comma-separated:
// `off = reminders, tasks`. An extension that is off is not discovered, so
// it has no rows, no view and no actions anywhere. Settings is never off:
// it is the way back to turning the others on.
const (
	Off      = "off"
	AlwaysOn = "settings"
)

// OffList is the extensions turned off, from the file.
func OffList() []string { return ParseOff(Get(Off, "")) }

// ParseOff reads the off value: names split on commas, spaces trimmed,
// empty ones and repeats dropped, and Settings never in it. A name that is
// not installed is kept; it may be installed later, or be in another
// directory, and dropping it would forget what the user wrote.
func ParseOff(value string) []string {
	var names []string
	seen := map[string]bool{}
	for _, n := range strings.Split(value, ",") {
		n = strings.TrimSpace(n)
		if n == "" || n == AlwaysOn || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names
}

// FormatOff is the value ParseOff reads back: the names, comma-separated.
func FormatOff(names []string) string { return strings.Join(ParseOff(strings.Join(names, ",")), ", ") }

// Dir is the config directory the file lives in.
func Dir() string {
	p := Path()
	if p == "" {
		return ""
	}
	return filepath.Dir(p)
}

// Get returns the value of key in the file, or fallback when the file or
// the key is not there. A file that cannot be read is the same as none.
func Get(key, fallback string) string {
	data, err := os.ReadFile(Path())
	if err != nil {
		return fallback
	}
	return get(string(data), key, fallback)
}

func get(text, key, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := parse(line)
		if ok && k == key {
			return v
		}
	}
	return fallback
}

// parse splits one line into key and value, or reports that it is not a
// setting (blank, a comment, or malformed).
func parse(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	k, v, found := strings.Cut(line, "=")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(k), strings.TrimSpace(v), true
}

// Set writes key = value into the file, replacing the key's line if it
// has one and appending it otherwise, and leaves every other line as it
// was. The file and its directory are created on first use.
func Set(key, value string) error {
	path := Path()
	if path == "" {
		return errors.New("settings: no home directory")
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(set(string(data), key, value)), 0o600)
}

func set(text, key, value string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}
	replaced := false
	for i, line := range lines {
		if k, _, ok := parse(line); ok && k == key {
			lines[i] = key + " = " + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, key+" = "+value)
	}
	return strings.Join(lines, "\n") + "\n"
}

// PreviewPercent is the preview width from the file, kept inside its
// range, or the default.
func PreviewPercent() int {
	return clampPreview(atoi(Get(Preview, ""), PreviewDefault))
}

// ClampPreview keeps a width inside PreviewMin..PreviewMax.
func ClampPreview(pct int) int { return clampPreview(pct) }

func clampPreview(pct int) int {
	if pct < PreviewMin {
		return PreviewMin
	}
	if pct > PreviewMax {
		return PreviewMax
	}
	return pct
}

func atoi(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}

// Editor is the editor to run, split on spaces so a line with
// flags works: the setting, else $EDITOR, else nano. The file to edit
// goes after it.
func Editor() []string {
	return editorCommand(Get(EditorKey, ""), os.Getenv("EDITOR"))
}

func editorCommand(setting, env string) []string {
	for _, line := range []string{setting, env} {
		if argv := strings.Fields(line); len(argv) > 0 {
			return argv
		}
	}
	return []string{EditorDefault}
}

// Edit runs the editor on file with the terminal attached, and returns
// when it exits. The caller runs under fzf's execute, which has handed
// over the whole terminal for the length of the run.
func Edit(file string) error {
	argv := append(Editor(), file)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
