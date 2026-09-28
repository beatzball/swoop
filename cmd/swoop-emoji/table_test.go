package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/usage"
)

// The table as shipped: every line parses, ids are unique, and the ones
// people look for first are there with the right names.
func TestTableParses(t *testing.T) {
	entries := load()
	if n := strings.Count(table, "\n"); len(entries) != n {
		t.Fatalf("%d lines, %d entries: a line does not parse", n, len(entries))
	}
	if len(entries) < 2000 {
		t.Fatalf("only %d entries", len(entries))
	}
	seen := map[string]bool{}
	groups := map[string]bool{}
	for _, e := range entries {
		if e.Char == "" || e.Name == "" || e.Group == "" {
			t.Errorf("an empty field: %+v", e)
		}
		if seen[key(e.Char)] {
			t.Errorf("%s twice", e.Char)
		}
		seen[key(e.Char)] = true
		groups[e.Group] = true
		if e.Tone != "" && !strings.Contains(e.Tone, "~") {
			t.Errorf("%s: a tone template without ~: %q", e.Name, e.Tone)
		}
	}
	for want, name := range map[string]string{"🚀": "rocket", "👍": "thumbs up", "🇯🇵": "flag: Japan", "€": "euro", "→": "right-pointing arrow"} {
		e, ok := find(entries, want)
		if !ok || e.Name != name {
			t.Errorf("%s: %+v", want, e)
		}
	}
	for _, g := range []string{"Smileys & Emotion", "Flags", "Arrows", "Math", "Currency", "Greek", "Box drawing"} {
		if !groups[g] {
			t.Errorf("no %s", g)
		}
	}
}

func TestWithTone(t *testing.T) {
	entries := load()
	up, _ := find(entries, "👍")
	if got := up.WithTone(3); got != "👍🏽" {
		t.Errorf("medium: %q", got)
	}
	if up.WithTone(0) != "👍" || up.WithTone(9) != "👍" {
		t.Error("0 or out of range is the emoji as it is")
	}
	// Two people, one tone for both.
	pair, _ := find(entries, "🧑‍🤝‍🧑")
	if got := pair.WithTone(1); got != "🧑🏻‍🤝‍🧑🏻" {
		t.Errorf("holding hands: %q", got)
	}
	rocket, _ := find(entries, "🚀")
	if rocket.Tone != "" || rocket.WithTone(5) != "🚀" {
		t.Error("a rocket takes no tone")
	}
}

func chars(es []Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Char)
	}
	return out
}

func TestSearch(t *testing.T) {
	entries := load()
	cases := []struct{ query, first string }{
		{"rocket", "🚀"},
		{"Rocket", "🚀"},
		{"heart", "❤️"},
		{"thumbs", "👍"},
		{"+1", "👍"},
		{"flag jap", "🇯🇵"},
		{"cmd", "⌘"},
		{"alpha", "α"},
		{"euro", "€"},
		{"🚀", "🚀"},
	}
	for _, c := range cases {
		got := search(entries, nil, c.query)
		if len(got) == 0 || got[0].Char != c.first {
			t.Errorf("%q: want %s first, got %v", c.query, c.first, chars(got[:min(len(got), 5)]))
		}
	}
	if got := search(entries, nil, "zzqxv"); len(got) != 0 {
		t.Errorf("nonsense matched %v", chars(got))
	}
	if got := search(entries, nil, "  "); len(got) != len(entries) {
		t.Error("an empty query is the whole table")
	}
	// Every word must match: "red" and "heart" together, not either.
	for _, e := range search(entries, nil, "red heart") {
		if !strings.Contains(e.Name+strings.Join(e.Keywords, " "), "red") {
			t.Errorf("%s %s has no red", e.Char, e.Name)
		}
	}
}

func TestCustomKeywords(t *testing.T) {
	extra := parseCustom("# mine\n😀 Yay zorblat\n🚀\nbad\n🚀 ship\n")
	if len(extra) != 2 || strings.Join(extra[key("🚀")], " ") != "ship" || strings.Join(extra[key("😀")], " ") != "yay zorblat" {
		t.Fatalf("parse: %v", extra)
	}
	got := search(load(), extra, "zorbl")
	if len(got) != 1 || got[0].Char != "😀" {
		t.Errorf("custom keyword: %v", chars(got))
	}
	// A keyword written with the presentation selector finds the emoji
	// stored without, and the other way round.
	extra = parseCustom("❤\U0000FE0F love2\n")
	if got := search(load(), extra, "love2"); len(got) != 1 || got[0].Name != "red heart" {
		t.Errorf("FE0F: %v", chars(got))
	}
}

// The view leads with the characters Enter was last used on, read from
// the usage log under their routed ids.
func TestRecentLeads(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, id := range []string{"ext/emoji/🎉", "ext/links/Google", "ext/emoji/🚀", "ext/emoji/emoji"} {
		if err := usage.Record(id, "emoji", ""); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
	if got := recent(); strings.Join(got, " ") != "🚀 🎉" {
		t.Fatalf("recent: %v", got)
	}
	out := captureView(t, "")
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "🚀\t") || !strings.HasPrefix(lines[1], "🎉\t") || !strings.Contains(lines[0], usage.Mark) {
		t.Errorf("recent rows first, marked: %q", lines[:3])
	}
}

func captureView(t *testing.T, query string) string {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = f
	err = view(query)
	os.Stdout = stdout
	// Closed before the read, and before the test's cleanup: Windows
	// refuses to remove a file that is still open.
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(f.Name())
	return string(data)
}

// The default tone comes from Settings and shows in the rows' icons.
func TestDefaultToneFromSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if defaultTone() != 0 {
		t.Fatal("unset is no tone")
	}
	if err := os.MkdirAll(filepath.Join(dir, "swoop"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "swoop", "config"), []byte("skin = 4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if defaultTone() != 4 {
		t.Fatal("skin = 4")
	}
	if out := captureView(t, "thumbs up"); !strings.HasPrefix(out, "👍\temoji\t👍🏾\t") {
		t.Errorf("the icon in the default tone, the id without: %q", strings.SplitN(out, "\n", 2)[0])
	}
}
