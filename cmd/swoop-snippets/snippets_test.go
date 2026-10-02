package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/tool"
)

// fixed is a clock, a clipboard and a UUID that never change, and counts
// how often the clipboard was read.
func fixed(clip string, reads *int) Env {
	return Env{
		Now: func() time.Time { return time.Date(2026, 9, 28, 14, 5, 9, 0, time.Local) },
		Clipboard: func() (string, error) {
			*reads++
			return clip, nil
		},
		UUID: func() string { return "00000000-0000-4000-8000-000000000000" },
	}
}

func TestFill(t *testing.T) {
	for _, c := range []struct {
		in, want string
		back     int
	}{
		{"plain", "plain", 0},
		{"on {date} at {time}", "on 2026-09-28 at 14:05", 0},
		{"got: {clipboard}", "got: pasted", 0},
		{"id {uuid}", "id 00000000-0000-4000-8000-000000000000", 0},
		{"<b>{cursor}</b>", "<b></b>", 4},
		{"{cursor}end", "end", 3},
		{"end{cursor}", "end", 0},
		// only the first {cursor} counts; the rest go
		{"a{cursor}b{cursor}c", "abc", 2},
		// characters, not bytes
		{"({cursor}é🚀)", "(é🚀)", 3},
		// braces that are not a placeholder stay, even next to one
		{"func() {}{date} {name} {{time}}", "func() {}2026-09-28 {name} {14:05}", 0},
		{"{Date} {date", "{Date} {date", 0},
	} {
		reads := 0
		got, back := Fill(c.in, fixed("pasted", &reads))
		if got != c.want || back != c.back {
			t.Errorf("Fill(%q) = %q, %d; want %q, %d", c.in, got, back, c.want, c.back)
		}
	}
}

// The clipboard is read once, only when asked for, and what it holds is
// not filled again.
func TestFillClipboard(t *testing.T) {
	reads := 0
	if got, _ := Fill("no clipboard here", fixed("x", &reads)); got != "no clipboard here" || reads != 0 {
		t.Errorf("read %d times for %q", reads, got)
	}
	got, _ := Fill("{clipboard} and {clipboard}", fixed("{date}", &reads))
	if got != "{date} and {date}" || reads != 1 {
		t.Errorf("got %q after %d reads", got, reads)
	}
	env := fixed("", &reads)
	env.Clipboard = func() (string, error) { return "", errors.New("no clipboard") }
	if got, _ := Fill("[{clipboard}]", env); got != "[]" {
		t.Errorf("a clipboard that fails fills in empty: %q", got)
	}
}

func TestNewUUID(t *testing.T) {
	v4 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	a, b := newUUID(), newUUID()
	if !v4.MatchString(a) || a == b {
		t.Errorf("not two fresh v4 UUIDs: %s %s", a, b)
	}
}

func TestParse(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Snippet
	}{
		{"Signature\nkeyword: ;sig\n\nBest,\nSam\n", Snippet{Name: "Signature", Keyword: ";sig", Text: "Best,\nSam"}},
		{"# Signature\nBest,\nSam", Snippet{Name: "Signature", Text: "Best,\nSam"}},
		{"Sig\r\nKeyword:  ;s \r\n\r\nline one\r\n\r\nline three\r\n", Snippet{Name: "Sig", Keyword: ";s", Text: "line one\n\nline three"}},
		{"Only a name", Snippet{Name: "Only a name"}},
		{"Trailing\n\ntext\n\n", Snippet{Name: "Trailing", Text: "text\n"}},
	} {
		if got := parse(c.in); got != c.want {
			t.Errorf("parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

// home points the config at a temp directory and returns the folder.
func home(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", d)
	// The heads cache goes under the user's cache directory, which is
	// under HOME on a Mac: keep it in the test's directory too.
	t.Setenv("HOME", d)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(d, "cache"))
	return filepath.Join(d, tool.Name(), "snippets")
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// heads reads names and keywords, skips what is not a snippet, and a
// missing folder is no snippets.
func TestHeads(t *testing.T) {
	d := home(t)
	if got, err := heads(); err != nil || len(got) != 0 {
		t.Fatalf("no folder: %v, %v", got, err)
	}
	write(t, d, "b.md", "Bee\nkeyword: bb\n\n"+strings.Repeat("long ", 1000))
	write(t, d, "a.md", "# Ay\ntext at once")
	write(t, d, "notes.txt", "Not a snippet\n")
	write(t, d, ".hidden.md", "Hidden\n")
	write(t, d, "empty.md", "")
	write(t, d, "long.md", strings.Repeat("n", 600)+"\nkeyword: k\n")
	got, err := heads()
	if err != nil {
		t.Fatal(err)
	}
	want := []Snippet{
		{File: "a.md", Name: "Ay"},
		{File: "b.md", Name: "Bee", Keyword: "bb"},
		{File: "long.md", Name: strings.Repeat("n", 600), Keyword: "k"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestReadRefusesPaths(t *testing.T) {
	home(t)
	for _, id := range []string{"", "../config", "a/b.md", "x.txt"} {
		if _, err := read(id); err == nil {
			t.Errorf("read(%q) should fail", id)
		}
	}
}

// Import writes a file per snippet that reads back the same; a second
// import of the same name replaces the file; two names with one slug get
// two files.
func TestImport(t *testing.T) {
	d := home(t)
	export := filepath.Join(t.TempDir(), "export.json")
	body := `[
		{"name": "Signature", "text": "Best,\nSam", "keyword": ";sig"},
		{"name": "Date stamp", "text": "{date} {time}"},
		{"name": "date/stamp", "text": "other"},
		{"name": "", "text": "no name"},
		{"name": "No text", "text": ""}
	]`
	if err := os.WriteFile(export, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := importJSON(export); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]Snippet{
		"signature.md":    {File: "signature.md", Name: "Signature", Keyword: ";sig", Text: "Best,\nSam"},
		"date-stamp.md":   {File: "date-stamp.md", Name: "Date stamp", Text: "{date} {time}"},
		"date-stamp-2.md": {File: "date-stamp-2.md", Name: "date/stamp", Text: "other"},
	} {
		got, err := read(file)
		if err != nil || got != want {
			t.Errorf("%s: %+v, %v; want %+v", file, got, err, want)
		}
	}
	if err := os.WriteFile(export, []byte(`[{"name": "Signature", "text": "Cheers"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := importJSON(export); err != nil {
		t.Fatal(err)
	}
	if got, _ := read("signature.md"); got.Text != "Cheers" || got.Keyword != "" {
		t.Errorf("not replaced: %+v", got)
	}
	entries, _ := os.ReadDir(d)
	if len(entries) != 3 {
		t.Errorf("want 3 files, have %d", len(entries))
	}
	if err := os.WriteFile(export, []byte(`{"not": "an array"}`), 0o600); err == nil {
		if importJSON(export) == nil {
			t.Error("an object is not an export")
		}
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Email Signature": "email-signature",
		"  ../etc/passwd": "etc-passwd",
		"Grüße":           "grüße",
		"???":             "snippet",
		"a: b":            "a-b",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// The cache gives the same heads as the files, sees a file edited in
// place, a file added and one removed, and is not used for another
// folder.
func TestHeadsCache(t *testing.T) {
	d := home(t)
	write(t, d, "a.md", "Ay\nkeyword: a\n\ntext")
	write(t, d, "b.md", "Bee\n")
	names := func() string {
		t.Helper()
		got, err := heads()
		if err != nil {
			t.Fatal(err)
		}
		var s []string
		for _, h := range got {
			s = append(s, h.Name+"/"+h.Keyword)
		}
		return strings.Join(s, " ")
	}
	if got := names(); got != "Ay/a Bee/" {
		t.Fatalf("first list: %s", got)
	}
	if _, err := os.Stat(cachePath()); err != nil {
		t.Fatalf("no cache written: %v", err)
	}
	if got := names(); got != "Ay/a Bee/" {
		t.Fatalf("from the cache: %s", got)
	}
	// In place, one byte longer, so the size changes even on a file
	// system whose times are coarse.
	write(t, d, "a.md", "Ayy\nkeyword: a\n\ntext")
	write(t, d, "c.md", "Sea\n")
	if err := os.Remove(filepath.Join(d, "b.md")); err != nil {
		t.Fatal(err)
	}
	if got := names(); got != "Ayy/a Sea/" {
		t.Fatalf("after edits: %s", got)
	}
	// Another folder with a file of the same name, size and time must
	// not get this folder's heads.
	other := filepath.Join(t.TempDir(), tool.Name(), "snippets")
	write(t, other, "a.md", "Zed\nkeyword: a\n\ntext")
	st, _ := os.Stat(filepath.Join(d, "a.md"))
	_ = os.Chtimes(filepath.Join(other, "a.md"), st.ModTime(), st.ModTime())
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(filepath.Dir(other)))
	if got := names(); got != "Zed/a" {
		t.Fatalf("another folder: %s", got)
	}
}

// Enter with no snippets writes the example and opens it; a second Enter
// keeps the file as it is. New snippet makes a file named by the time.
func TestEditWritesOnceAndLands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("EDITOR", "true")
	land := filepath.Join(t.TempDir(), "land")
	t.Setenv("SWOOP_LAND", land)
	if err := run(newID, ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(land)
	if err != nil || string(got) != "signature.md\n" {
		t.Fatalf("the landing should name the example's file: %q %v", got, err)
	}
	p := filepath.Join(dir(), "signature.md")
	if err := os.WriteFile(p, []byte("Signature\n\nEdited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(newID, ""); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(p); string(data) != "Signature\n\nEdited\n" {
		t.Errorf("a second Enter must not write over the edited example: %q", data)
	}
	if err := run("signature.md", "new"); err != nil {
		t.Fatal(err)
	}
	have, _ := heads()
	if len(have) != 2 {
		t.Fatalf("New snippet should add one file: %+v", have)
	}
	if out := capture(t, func() error { return actions("signature.md") }); !strings.Contains(out, "edit\tterminal\t") {
		t.Errorf("ctrl-k should offer Edit as a terminal row: %q", out)
	}
	t.Setenv("EDITOR", "false")
	if err := run("signature.md", "edit"); err == nil {
		t.Error("Edit runs the editor: with `false` as the editor it should fail")
	}
}

// The Snippets row is always at the root; its pane leads with New
// snippet and filters by name or keyword.
func TestSnippetsRowAndView(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out := capture(t, func() error { return list() })
	if !strings.HasPrefix(out, viewID+"\tview\t") {
		t.Fatalf("the Snippets row should lead the root rows: %q", out)
	}
	if err := put([]Snippet{{Name: "Greeting", Keyword: ";hi", Text: "Hello"}, {Name: "Address", Text: "1 Main St"}}); err != nil {
		t.Fatal(err)
	}
	out = capture(t, func() error { return view("") })
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], newID+"\tterminal\t") {
		t.Fatalf("New snippet, then two: %q", out)
	}
	out = capture(t, func() error { return view(";hi") })
	if lines := strings.Split(strings.TrimSpace(out), "\n"); len(lines) != 2 || !strings.Contains(lines[1], "Greeting") {
		t.Fatalf("a keyword filters the pane: %q", out)
	}
	// The letters in order find a name (#201), and New snippet stays first.
	out = capture(t, func() error { return view("grtg") })
	if lines := strings.Split(strings.TrimSpace(out), "\n"); len(lines) != 2 || !strings.HasPrefix(lines[0], newID+"\t") || !strings.Contains(lines[1], "Greeting") {
		t.Fatalf("letters in order: %q", out)
	}
}

// capture runs fn with stdout in a file and returns what it printed.
func capture(t *testing.T, fn func() error) string {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = f
	err = fn()
	os.Stdout = stdout
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(f.Name())
	return string(data)
}
