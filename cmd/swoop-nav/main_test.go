package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/nav"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/tool"
)

func TestMatchingNeedsEveryWord(t *testing.T) {
	items := []protocol.Item{{Title: "Left Half"}, {Title: "Left Third"}, {Title: "Right Half"}}
	// "lfhalf" is letters in order, with others in between (#201).
	cases := map[string]int{"": 3, "left": 2, "HALF left": 1, "  half  ": 2, "top": 0, "lfhalf": 1}
	for q, want := range cases {
		if got := len(matching(items, q)); got != want {
			t.Errorf("matching(%q) kept %d, want %d", q, got, want)
		}
	}
}

// A keyword's rows, Links' for one, filter on the subtitle too, and the
// best match comes first: a title ahead of a subtitle.
func TestMatchingRanks(t *testing.T) {
	items := []protocol.Item{
		{ID: "1", Title: "Mail", Subtitle: "https://mail.example.com"},
		{ID: "2", Title: "Docs", Subtitle: "https://example.com/docs"},
		{ID: "3", Title: "Example", Subtitle: "https://www.example.org"},
	}
	var got string
	for _, it := range matching(items, "example") {
		got += it.ID
	}
	if got != "312" {
		t.Errorf("got %s, want 312", got)
	}
	if got := matching(items, "exmpl"); len(got) != 3 || got[0].ID != "3" {
		t.Errorf("letters in order: %+v", got)
	}
}

// fakeExt writes a bash extension called name into dir, with files
// beside it.
func fakeExt(t *testing.T, dir, name, body string, files map[string]string) {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, name), []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	for f, text := range files {
		if err := os.WriteFile(filepath.Join(d, f), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Run it once, with no verb, so its first start is not inside a
	// list's two seconds: on a busy machine the first start of a new
	// program has taken longer than that.
	_ = exec.Command(filepath.Join(d, name)).Run()
}

// An extension with a cache file is listed once per launch, with no text,
// and its rows kept; the rest are asked every time, with the text. Turned
// off, its kept rows are gone; turned on again, they are back without
// another list. One turned on for the first time mid-run is listed then.
func TestRootRowsListsOnceAndKeeps(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash scripts do not run as executables on Windows")
	}
	dir, cfg, tmp := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("SWOOP_EXTENSIONS", dir)
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv(envOnce, filepath.Join(tmp, "once"))
	t.Setenv(envIcons, "")
	count := filepath.Join(tmp, "count")
	fakeExt(t, dir, "kept", `[ "$1" = list ] || exit 0; echo "${2:-none}" >> '`+count+`'; printf 'a\tthing\t\tKept A\t\n'`, map[string]string{"cache": "run\n"})
	fakeExt(t, dir, "late", `printf 'b\tthing\t\tLate B\t\n'`, map[string]string{"cache": "run\n"})
	fakeExt(t, dir, "live", `printf 'c\tthing\t\tLive %s\t\n' "${2:-none}"`, nil)
	off := func(names string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(cfg, tool.Name()), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfg, tool.Name(), "config"), []byte("off = "+names+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	titles := func(query string) string {
		var ts []string
		for _, it := range rootRows(query) {
			ts = append(ts, it.ID+"="+it.Title)
		}
		return strings.Join(ts, ", ")
	}

	off("late")
	if got, want := titles(""), "ext/kept/a=Kept A, ext/live/c=Live none"; got != want {
		t.Fatalf("first rows: %s, want %s", got, want)
	}
	if got, want := titles("x"), "ext/kept/a=Kept A, ext/live/c=Live x"; got != want {
		t.Fatalf("rows for x: %s, want %s", got, want)
	}
	off("late, kept")
	if got, want := titles("x"), "ext/live/c=Live x"; got != want {
		t.Fatalf("kept turned off: %s, want %s", got, want)
	}
	off("")
	if got, want := titles("x"), "ext/kept/a=Kept A, ext/late/b=Late B, ext/live/c=Live x"; got != want {
		t.Fatalf("all on: %s, want %s", got, want)
	}
	// Listed once in all that, and with no text.
	data, err := os.ReadFile(count)
	if err != nil || string(data) != "none\n" {
		t.Fatalf("kept was listed %q, %v; want once, with no text", data, err)
	}
}

// The key for an extension's pane is the first it claims and has: not one
// an earlier extension has, not one nobody may claim, and none at all for
// an extension that is off, claims nothing, or is not there. The frame's
// menu shows Settings only when this names a key.
func TestKeyFor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash scripts do not run as executables on Windows")
	}
	dir, cfg := t.TempDir(), t.TempDir()
	t.Setenv("SWOOP_EXTENSIONS", dir)
	t.Setenv("XDG_CONFIG_HOME", cfg)
	fakeExt(t, dir, "ask", "", map[string]string{"key": "tab ask Ask\n"})
	fakeExt(t, dir, "plain", "", nil)
	fakeExt(t, dir, "prefs", "", map[string]string{"key": "ctrl-c prefs\ntab prefs\nalt-, prefs Prefs\n"})
	fakeExt(t, dir, "shadow", "", map[string]string{"key": "tab shadow\n"})
	want := map[string]string{"ask": "tab", "prefs": "alt-,", "plain": "", "shadow": "", "absent": ""}
	for name, key := range want {
		if got := keyFor(name); got != key {
			t.Errorf("keyFor(%q) = %q, want %q", name, got, key)
		}
	}
	if err := os.MkdirAll(filepath.Join(cfg, tool.Name()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, tool.Name(), "config"), []byte("off = ask, prefs\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Off, prefs has no key; and with ask off, Tab is the next one's.
	for name, key := range map[string]string{"prefs": "", "ask": "", "shadow": "tab"} {
		if got := keyFor(name); got != key {
			t.Errorf("with ask and prefs off, keyFor(%q) = %q, want %q", name, got, key)
		}
	}
}

// With no extensions at all the root is an empty list, not an error.
func TestRootRowsWithNoExtensions(t *testing.T) {
	t.Setenv("SWOOP_EXTENSIONS", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv(envOnce, filepath.Join(t.TempDir(), "once"))
	items, err := rows(&nav.State{}, "")
	if err != nil || len(items) != 0 {
		t.Fatalf("rows = %+v, %v; want none and no error", items, err)
	}
}

// The rows that asked for icons go through swoop-icons, the rest are not
// touched, and the order holds.
func TestPicturesSendsOnlyTheRowsThatAsked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash scripts do not run as executables on Windows")
	}
	dir, bin := t.TempDir(), t.TempDir()
	fakeExt(t, dir, "pics", "exit 0\n", map[string]string{"cache": "run\n", "icons": "id\n"})
	fakeExt(t, dir, "plain", "exit 0\n", map[string]string{"cache": "run\n"})
	// A stand-in for swoop-icons: it marks the icon of each row it is
	// given, and says where it was told to put the pictures.
	if err := os.WriteFile(filepath.Join(bin, "swoop-icons"), []byte("#!/usr/bin/env bash\necho \"$2\" > '"+filepath.Join(bin, "out")+"'\nawk -F'\\t' -v OFS='\\t' '{ $3 = \"PIC\"; print }'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	items := []protocol.Item{
		{ID: "ext/plain/1", Kind: "thing", Icon: "p", Title: "One"},
		{ID: "ext/pics//some/file", Kind: "thing", Icon: "g", Title: "Two"},
		{ID: "ext/plain/3", Kind: "thing", Icon: "p", Title: "Three"},
	}
	got := pictures(items, ext.Discover([]string{dir}), filepath.Join(bin, "pictures"))
	var icons []string
	for _, it := range got {
		icons = append(icons, it.Icon)
	}
	if strings.Join(icons, ",") != "p,PIC,p" {
		t.Fatalf("icons: %v", icons)
	}
	out, _ := os.ReadFile(filepath.Join(bin, "out"))
	if strings.TrimSpace(string(out)) != filepath.Join(bin, "pictures") {
		t.Fatalf("swoop-icons was sent the pictures to %q", out)
	}
}
