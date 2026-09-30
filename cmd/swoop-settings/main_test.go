package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
	"github.com/beatzball/swoop/internal/width"
)

func TestHotkeyChoicesTakeWhatWasTyped(t *testing.T) {
	cs := hotkeyChoices("")
	if len(cs) != 6 || cs[0].value != "alt+shift+space" {
		t.Fatalf("the fixed list: %+v", cs)
	}
	cs = hotkeyChoices("ctrl+alt+k")
	if cs[0].value != "ctrl+alt+k" || !cs[0].typed {
		t.Fatalf("a typed key comes first: %+v", cs[0])
	}
	cs = hotkeyChoices("alt+space")
	if len(cs) != 6 {
		t.Fatalf("a typed key that is on the list is not added again: %d", len(cs))
	}
	for _, bad := range []string{"k", "meta+k", "alt+", "just words"} {
		if looksLikeHotkey(bad) {
			t.Errorf("%q should not look like a hotkey", bad)
		}
	}
}

func TestRunSetsAndClamps(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := run("preview=95"); err != nil {
		t.Fatal(err)
	}
	if got := settings.PreviewPercent(); got != settings.PreviewMax {
		t.Fatalf("preview clamped: %d", got)
	}
	if err := run("web=on"); err != nil {
		t.Fatal(err)
	}
	if got := settings.Get(settings.Web, "off"); got != "on" {
		t.Fatalf("web: %q", got)
	}
	if err := run("nosuch=1"); err == nil {
		t.Fatal("an unknown key is refused")
	}
	if err := run("settings"); err != nil {
		t.Fatalf("a row with nothing to run is fine: %v", err)
	}
}

func TestSkinTone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := find(settings.Skin)
	if s == nil || len(s.choices("")) != 6 {
		t.Fatal("six skin tones")
	}
	if !strings.Contains(s.value(), "None") {
		t.Errorf("unset: %q", s.value())
	}
	if err := run("skin=3"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.value(), "Medium") {
		t.Errorf("after skin=3: %q", s.value())
	}
}

func TestExtensionsTurnOnAndOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	for name, body := range map[string]string{
		"alpha":    "#!/usr/bin/env bash\n# alpha: the first one. More words.\n",
		"beta":     "#!/usr/bin/env bash\n# beta: wraps in the middle,\n# of a sentence.\n",
		"settings": "#!/usr/bin/env bash\n# settings: the way back.\n",
		"gamma":    "not a script\n",
	} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SWOOP_EXTENSIONS", dir)
	marks := func(query string) string {
		var out []string
		for _, it := range extensionRows(query) {
			if it.Kind != "toggle" || it.ID != extensionPrefix+it.Title {
				t.Fatalf("a row: %+v", it)
			}
			mark := "on"
			if it.Icon == iconOff {
				mark = "off"
			}
			out = append(out, it.Title+"="+mark+":"+it.Subtitle)
		}
		return strings.Join(out, "|")
	}
	want := "alpha=on:the first one|beta=on:wraps in the middle|gamma=on:|settings=on:always on · the way back"
	if got := marks(""); got != want {
		t.Fatalf("all on:\n%s\nwant\n%s", got, want)
	}
	if err := run("extension/alpha"); err != nil {
		t.Fatal(err)
	}
	if err := run("extension/gamma"); err != nil {
		t.Fatal(err)
	}
	if got := settings.Get(settings.Off, ""); got != "alpha, gamma" {
		t.Fatalf("off = %q", got)
	}
	if got := marks("a"); got != "alpha=off:off · the first one|beta=on:wraps in the middle|gamma=off:off" {
		t.Fatalf("after two flips, filtered by a: %s", got)
	}
	if got := extensionsValue(); got != "2 on, 2 off" {
		t.Fatalf("the Extensions row: %q", got)
	}
	// Settings cannot be turned off, and a second flip turns one back on.
	if err := run("extension/settings"); err != nil {
		t.Fatal(err)
	}
	if err := run("extension/alpha"); err != nil {
		t.Fatal(err)
	}
	if got := settings.Get(settings.Off, ""); got != "gamma" {
		t.Fatalf("off = %q", got)
	}
}

func TestEditorChoicesAreWhatIsOnPath(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"nvim", "emacs"} {
		if runtime.GOOS == "windows" {
			// LookPath on Windows finds a program only by an extension
			// from PATHEXT.
			name += ".exe"
		}
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	var got []string
	for _, c := range editorChoices("") {
		got = append(got, c.value)
	}
	if strings.Join(got, "|") != "|nvim|emacs -nw" {
		t.Fatalf("Default and the editors on PATH: %q", got)
	}
	if cs := editorChoices("nvim  --clean"); cs[0].value != "nvim --clean" || cs[0].note != "what you typed" {
		t.Fatalf("a typed command on PATH comes first: %+v", cs[0])
	}
	if cs := editorChoices("emacs -nw"); len(cs) != 3 {
		t.Fatalf("a typed command already listed is not added again: %+v", cs)
	}
	if cs := editorChoices("notthere"); len(cs) != 3 {
		t.Fatalf("a typed command not on PATH is not offered: %+v", cs)
	}
}

// Every row the pane shows has an icon exactly one cell wide, so the
// titles after it sit in one column (#150): the root row, the settings,
// the extensions on and off, and every setting's choices, with and
// without something typed.
func TestEveryIconIsOneCell(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	for _, name := range []string{"alpha", "settings"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SWOOP_EXTENSIONS", dir)
	check := func(pane string, items []protocol.Item) {
		t.Helper()
		if len(items) == 0 {
			t.Errorf("%s: no rows", pane)
		}
		for _, it := range items {
			if w := width.String(it.Icon); w != 1 {
				t.Errorf("%s: %q has icon %q, %d cells wide", pane, it.Title, it.Icon, w)
			}
		}
	}
	check("root", []protocol.Item{root})
	get := func(id, query string) []protocol.Item {
		t.Helper()
		items, err := rows(id, query)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}
	check("settings", get("settings", ""))
	check("extensions", get(extensionsID, ""))
	if err := run("extension/alpha"); err != nil {
		t.Fatal(err)
	}
	check("extensions, one off", get(extensionsID, ""))
	for _, s := range all {
		check(s.key, get(s.key, ""))
	}
	for key, typed := range map[string]string{
		settings.Hotkey: "ctrl+alt+k",
		settings.AIURL:  "http://localhost:8080/v1",
	} {
		check(key+", typed", get(key, typed))
	}
}

// A pane that takes a value of the user's own shows the row made from
// what was typed, whatever that row's title and note say (#160): the API
// key's row is titled "Use what you typed" and noted "N characters", and
// a filter on either hid it.
func TestTypedValueRowIsShown(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	bin := t.TempDir()
	name := "myed"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	typed := map[string]string{
		settings.Hotkey:    "ctrl+alt+k",
		settings.AI:        "openai:some-model",
		settings.AIURL:     "http://localhost:8080/v1",
		settings.AIKey:     "sk-test-0123456789",
		settings.EditorKey: "myed --clean",
	}
	for _, s := range all {
		text, ok := typed[s.key]
		// A setting whose choices can carry a typed row must be in the
		// map, so a new one cannot go untested.
		var makes bool
		for _, probe := range typed {
			for _, c := range s.choices(probe) {
				makes = makes || c.typed
			}
		}
		if makes != ok {
			t.Errorf("%s: makes a typed row: %v, in this test: %v", s.key, makes, ok)
		}
		if !ok {
			continue
		}
		items, err := rows(s.key, text)
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, it := range items {
			found = found || it.ID == s.key+"="+text
		}
		if !found {
			t.Errorf("%s: typed %q, and no row sets it: %+v", s.key, text, items)
		}
	}
}
