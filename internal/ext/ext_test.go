package ext

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/tool"
)

func TestRoute(t *testing.T) {
	cases := []struct {
		id, name, raw string
		ok            bool
	}{
		{"ext/system/sleep", "system", "sleep", true},
		{"ext/system/a/b/c", "system", "a/b/c", true},
		{"/Applications/Safari.app", "", "", false},
		{"ext/", "", "", false},
		{"ext/system", "", "", false},
		{"ext//x", "", "", false},
		{"ext/system/", "", "", false},
	}
	for _, c := range cases {
		name, raw, ok := Route(c.id)
		if name != c.name || raw != c.raw || ok != c.ok {
			t.Errorf("Route(%q) = %q, %q, %v; want %q, %q, %v", c.id, name, raw, ok, c.name, c.raw, c.ok)
		}
	}
}

// fake writes a bash extension called name into dir and returns its path.
// TestMain gives every list ten seconds: the real two are for a launcher
// that must not wait, and a test machine busy compiling has tripped it.
// TestListTimeoutIsAnError sets its own short limit.
func TestMain(m *testing.M) {
	listTimeout = 10 * time.Second
	// The machine's own settings file must not reach the tests: an off
	// list there would hide the fakes.
	cfg, err := os.MkdirTemp("", "ext-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", cfg)
	code := m.Run()
	os.RemoveAll(cfg)
	os.Exit(code)
}

func fake(t *testing.T, dir, name, body string) string {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(d, name)
	if err := os.WriteFile(exe, []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bash scripts do not run as executables on Windows")
	}
}

func TestDiscover(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "beta", "echo")
	fake(t, dir, "alpha", "echo")
	// A directory with no executable of its own name is not an extension.
	if err := os.MkdirAll(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A file at the top level is ignored too.
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An executable that is not executable is skipped.
	fake(t, dir, "gamma", "echo")
	if err := os.Chmod(filepath.Join(dir, "gamma", "gamma"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Discover([]string{dir, "/nonexistent/dir"})
	var names []string
	for _, e := range got {
		names = append(names, e.Name)
	}
	if strings.Join(names, ",") != "alpha,beta" {
		t.Fatalf("Discover found %v, want alpha,beta", names)
	}
	if got[0].Dir != filepath.Join(dir, "alpha") || got[0].Exe != filepath.Join(dir, "alpha", "alpha") {
		t.Fatalf("wrong paths: %+v", got[0])
	}
}

func TestDiscoverFirstDirWins(t *testing.T) {
	skipOnWindows(t)
	a, b := t.TempDir(), t.TempDir()
	fake(t, a, "same", "echo from-a")
	fake(t, b, "same", "echo from-b")
	got := Discover([]string{a, b})
	if len(got) != 1 || got[0].Dir != filepath.Join(a, "same") {
		t.Fatalf("want the first directory's copy only, got %+v", got)
	}
}

func TestDiscoverSkipsWhatIsOff(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	for _, n := range []string{"alpha", "beta", "gamma", "settings"} {
		fake(t, dir, n, "echo")
	}
	names := func(exts []Extension) string {
		var ns []string
		for _, e := range exts {
			ns = append(ns, e.Name)
		}
		return strings.Join(ns, ",")
	}
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	if got := names(Discover([]string{dir})); got != "alpha,beta,gamma,settings" {
		t.Fatalf("nothing off: %s", got)
	}
	// Spaces, an unknown name, and settings, which cannot be turned off.
	if err := os.MkdirAll(filepath.Join(cfg, tool.Name()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, tool.Name(), "config"), []byte("off =  gamma , nosuch, settings,alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := names(Discover([]string{dir})); got != "beta,settings" {
		t.Fatalf("off = gamma, alpha: %s", got)
	}
	if got := names(DiscoverAll([]string{dir})); got != "alpha,beta,gamma,settings" {
		t.Fatalf("DiscoverAll lists the off ones too: %s", got)
	}
	if got := names(skip(DiscoverAll([]string{dir}), nil)); got != "alpha,beta,gamma,settings" {
		t.Fatalf("an empty list skips nothing: %s", got)
	}
}

func TestListPrefixesAndSkipsBadLines(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "hello", `
case "$1" in
  list)
    printf 'one\tcommand\t*\tOne\tfirst\n'
    printf 'this line is not a row\n'
    printf 'two\tcommand\t*\tTwo\t\n'
    printf 'dir=%s\n' "$SWOOP_EXT_DIR" >&2
    ;;
esac`)
	e := Discover([]string{dir})[0]
	items, err := e.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 rows, got %d: %+v", len(items), items)
	}
	if items[0].ID != "ext/hello/one" || items[1].ID != "ext/hello/two" {
		t.Fatalf("ids not prefixed: %q, %q", items[0].ID, items[1].ID)
	}
	name, raw, ok := Route(items[0].ID)
	if !ok || name != "hello" || raw != "one" {
		t.Fatalf("Route does not undo the prefix: %q %q %v", name, raw, ok)
	}
}

func TestListFailureAndTimeoutAreErrors(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "broken", `echo "no such thing" >&2; exit 3`)
	e := Discover([]string{dir})[0]
	if _, err := e.List(""); err == nil || !strings.Contains(err.Error(), "no such thing") {
		t.Fatalf("want the extension's stderr in the error, got %v", err)
	}
}

func TestListEachKeepsTheGoodOnes(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "good", `[ "$1" = list ] && printf 'x\tcommand\t\tGood\t\n'; exit 0`)
	fake(t, dir, "bad", `exit 1`)
	items, names, errs := ListEach(Discover([]string{dir}), func(e Extension) ([]protocol.Item, error) { return e.List("") })
	if len(items) != 1 || items[0].Title != "Good" {
		t.Fatalf("want only the good extension's row, got %+v", items)
	}
	if len(names) != 1 || names[0] != "good" {
		t.Fatalf("want only the good extension named, got %v", names)
	}
	var le *LateError
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "bad: list") || errors.As(errs[0], &le) {
		t.Fatalf("want the bad extension's error, and not as a late one, got %v", errs)
	}
}

func TestPreviewAndRun(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	fake(t, dir, "act", `
case "$1" in
  preview) echo "preview of $2" ;;
  run) echo "$2" > "`+marker+`" ;;
esac`)
	e := Discover([]string{dir})[0]
	var out bytes.Buffer
	if err := e.Preview("thing", &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "preview of thing" {
		t.Fatalf("preview output: %q", out.String())
	}
	if err := e.Run("thing", ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(marker)
	if err != nil || strings.TrimSpace(string(got)) != "thing" {
		t.Fatalf("run did not receive the raw id: %q %v", got, err)
	}
}

func TestActionsAndRunWithAction(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	fake(t, dir, "clip", `
case "$1" in
  actions) printf 'copy\taction\t\tCopy\tEnter\n'; printf 'delete\trefresh\t\tDelete\t\n' ;;
  run) echo "$2 $3" > "`+marker+`" ;;
esac`)
	e := Discover([]string{dir})[0]
	acts := e.Actions("17")
	if len(acts) != 2 || acts[0].ID != "copy" || acts[1].ID != "delete" || acts[1].Kind != "refresh" {
		t.Fatalf("actions wrong: %+v", acts)
	}
	if err := e.Run("17", "delete"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(marker)
	if strings.TrimSpace(string(got)) != "17 delete" {
		t.Fatalf("run did not get id and action: %q", got)
	}
	fake(t, dir, "plain", `exit 0`)
	if acts := Discover([]string{dir})[1].Actions("x"); len(acts) != 0 {
		t.Fatalf("an extension without the verb must have no actions: %+v", acts)
	}
}

func TestViewPassesIDAndQuery(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "words", `
case "$1" in
  view) printf '%s\tword\t\t%s\t\n' "$2-$3" "got $2 with ${3:-nothing}" ;;
esac`)
	e := Discover([]string{dir})[0]
	items, err := e.View("define", "de")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "ext/words/define-de" || items[0].Title != "got define with de" {
		t.Fatalf("view rows wrong: %+v", items)
	}
	items, err = e.View("define", "")
	if err != nil || items[0].Title != "got define with nothing" {
		t.Fatalf("empty query should be omitted: %+v %v", items, err)
	}
}

func TestListTimeoutIsAnError(t *testing.T) {
	skipOnWindows(t)
	old := listTimeout
	listTimeout = 200 * time.Millisecond
	defer func() { listTimeout = old }()
	dir := t.TempDir()
	fake(t, dir, "slow", `sleep 5; printf 'x\tcommand\t\tSlow\t\n'`)
	e := Discover([]string{dir})[0]
	_, err := e.List("")
	var le *LateError
	if !errors.As(err, &le) || !strings.Contains(err.Error(), "slow: list took longer than 200ms") {
		t.Fatalf("want a late error, got %v", err)
	}
}

// The list kept for the run has the start's limit, not a keystroke's: one
// that outlasts a keystroke's limit is still taken as the launcher starts.
func TestListStartHasTheStartsLimit(t *testing.T) {
	skipOnWindows(t)
	oldList, oldStart := listTimeout, startTimeout
	listTimeout, startTimeout = 200*time.Millisecond, 10*time.Second
	defer func() { listTimeout, startTimeout = oldList, oldStart }()
	dir := t.TempDir()
	fake(t, dir, "slow", `sleep 0.5; printf 'x\tcommand\t\tSlow %s\t\n' "${2:-no text}"`)
	e := Discover([]string{dir})[0]
	var le *LateError
	if _, err := e.List(""); !errors.As(err, &le) {
		t.Fatalf("a keystroke's list: want a late error, got %v", err)
	}
	items, err := e.ListStart()
	if err != nil || len(items) != 1 || items[0].Title != "Slow no text" {
		t.Fatalf("the start's list: %+v, %v", items, err)
	}
}

func TestSplitKeyword(t *testing.T) {
	cases := []struct {
		query, kw, rest string
		ok              bool
	}{
		{"def ap", "def", "ap", true},
		{"def ", "def", "", true},
		{"def   ap  x", "def", "ap  x", true},
		{"win left half", "win", "left half", true},
		// The word alone is still on its way to an app's name.
		{"def", "", "", false},
		{"", "", "", false},
		// A leading space is no keyword.
		{" def ap", "", "", false},
	}
	for _, c := range cases {
		kw, rest, ok := SplitKeyword(c.query)
		if kw != c.kw || rest != c.rest || ok != c.ok {
			t.Errorf("SplitKeyword(%q) = %q, %q, %v; want %q, %q, %v", c.query, kw, rest, ok, c.kw, c.rest, c.ok)
		}
	}
}

func TestByKeywordReadsTheFileBesideTheExtension(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "define", "exit 0\n")
	fake(t, dir, "plain", "exit 0\n")
	fake(t, dir, "second", "exit 0\n")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name, KeywordFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Only the first word of the first line counts.
	write("define", "  def  extra\nignored\n")
	write("second", "DEF\n")
	exts := Discover([]string{dir})
	if got := exts[0].Keyword(); got != "def" {
		t.Fatalf("Keyword() = %q", got)
	}
	if got := exts[1].Keyword(); got != "" {
		t.Fatalf("no file is no keyword: %q", got)
	}
	// Case is ignored, and the first by name wins a shared keyword.
	e, ok := ByKeyword(exts, "Def")
	if !ok || e.Name != "define" {
		t.Fatalf("ByKeyword(Def) = %+v, %v", e, ok)
	}
	if _, ok := ByKeyword(exts, "plain"); ok {
		t.Fatal("a name is not a keyword")
	}
}

// TestBundledKeywords holds the keywords the repository's extensions ship
// with, so a missing or renamed file is caught here and not by a user.
func TestBundledKeywords(t *testing.T) {
	skipOnWindows(t)
	want := map[string]string{
		"ai": "ai", "calc": "calc", "clipboard": "clip", "define": "def",
		"emoji": "emoji", "files": "files", "links": "links", "notes": "notes",
		"reminders": "rem", "settings": "settings", "snippets": "snip",
		"stats": "stats", "tasks": "tasks", "window": "win",
	}
	got := map[string]string{}
	for _, e := range DiscoverAll([]string{filepath.Join("..", "..", "extensions")}) {
		if k := e.Keyword(); k != "" {
			got[e.Name] = k
		}
	}
	if len(got) != len(want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	for name, k := range want {
		if got[name] != k {
			t.Errorf("%s: keyword %q, want %q", name, got[name], k)
		}
	}
}

// The files beside an extension that ask the launcher for something: only
// the first word of the first line counts, and icons need the cache.
func TestOnceAndIconsReadTheFilesBesideTheExtension(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	for _, n := range []string{"both", "cached", "icons", "plain", "other"} {
		fake(t, dir, n, "exit 0\n")
	}
	write := func(name, file, body string) {
		if err := os.WriteFile(filepath.Join(dir, name, file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("both", CacheFile, "  run  extra\nignored\n")
	write("both", IconsFile, "id\n")
	write("cached", CacheFile, "run\n")
	// Pictures go to the terminal once, at the start: no cache, no icons.
	write("icons", IconsFile, "id\n")
	// A word this version does not know asks for nothing.
	write("other", CacheFile, "day\n")
	write("other", IconsFile, "id\n")
	want := map[string][2]bool{
		"both": {true, true}, "cached": {true, false}, "icons": {false, false},
		"plain": {false, false}, "other": {false, false},
	}
	for _, e := range Discover([]string{dir}) {
		if got := [2]bool{e.Once(), e.IconsByID()}; got != want[e.Name] {
			t.Errorf("%s: Once, IconsByID = %v, want %v", e.Name, got, want[e.Name])
		}
	}
}

// TestBundledOnce holds which of the repository's extensions are listed
// once per launch, so a missing file is caught here and not as a slow
// keystroke.
func TestBundledOnce(t *testing.T) {
	skipOnWindows(t)
	var got []string
	for _, e := range DiscoverAll([]string{filepath.Join("..", "..", "extensions")}) {
		if e.Once() {
			got = append(got, e.Name)
			if !e.IconsByID() {
				t.Errorf("%s: no icons file", e.Name)
			}
		}
	}
	if strings.Join(got, ",") != "apps" {
		t.Fatalf("listed once: %v, want apps", got)
	}
}
