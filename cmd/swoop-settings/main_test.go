package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/settings"
)

func TestHotkeyChoicesTakeWhatWasTyped(t *testing.T) {
	cs := hotkeyChoices("")
	if len(cs) != 6 || cs[0].value != "alt+shift+space" {
		t.Fatalf("the fixed list: %+v", cs)
	}
	cs = hotkeyChoices("ctrl+alt+k")
	if cs[0].value != "ctrl+alt+k" || cs[0].note != "what you typed" {
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
