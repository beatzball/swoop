// Package settings reads and writes ~/.config/swoop/config, the one file
// for the launcher's own choices: what the user changed with a key and
// wants to find the same way next time. One `key = value` per line,
// `#` starts a comment, unknown keys are kept as they are. The frame's
// own file, shell-mac.json, is separate: it belongs to one OS.
package settings

import (
	"errors"
	"os"
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
