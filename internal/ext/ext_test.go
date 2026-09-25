package ext

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

func TestListAllKeepsTheGoodOnes(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "good", `[ "$1" = list ] && printf 'x\tcommand\t\tGood\t\n'; exit 0`)
	fake(t, dir, "bad", `exit 1`)
	items := ListAll(Discover([]string{dir}), "")
	if len(items) != 1 || items[0].Title != "Good" {
		t.Fatalf("want only the good extension's row, got %+v", items)
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
