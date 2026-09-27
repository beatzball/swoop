package main

import (
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
