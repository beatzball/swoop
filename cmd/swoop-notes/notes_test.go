package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestTitle(t *testing.T) {
	for _, c := range []struct{ first, file, want string }{
		{"# Groceries", "groceries.md", "Groceries"},
		{"## Deep heading  ", "x.md", "Deep heading"},
		{"Plain first line", "x.md", "Plain first line"},
		{"   ", "blank-start.md", "blank-start"},
		{"#", "just-a-hash.md", "just-a-hash"},
		{"", "empty.md", "empty"},
	} {
		if got := Title(c.first, c.file); got != c.want {
			t.Errorf("Title(%q, %q) = %q, want %q", c.first, c.file, got, c.want)
		}
	}
	if got := Title(firstLine("# One\r\nTwo\n"), "x.md"); got != "One" {
		t.Errorf("only the first line, without its CR: %q", got)
	}
}

// note writes a file into d, changed at the given minute past a fixed
// hour, so the order does not depend on how fast the test writes.
func note(t *testing.T, d, name, body string, minute int) {
	t.Helper()
	if err := os.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 28, 12, minute, 0, 0, time.UTC)
	if err := os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

func files(notes []Note) []string {
	var out []string
	for _, n := range notes {
		out = append(out, n.File)
	}
	return out
}

func TestOrder(t *testing.T) {
	d := t.TempDir()
	note(t, d, "old.md", "# Old\n", 1)
	note(t, d, "new.md", "# New\n", 3)
	note(t, d, "b-tie.md", "# B\n", 2)
	note(t, d, "a-tie.md", "# A\n", 2)
	note(t, d, "not-a-note.txt", "x", 9)
	note(t, d, ".hidden.md", "x", 9)
	if err := os.Mkdir(filepath.Join(d, deletedDir), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := find(d, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"new.md", "a-tie.md", "b-tie.md", "old.md"}; !reflect.DeepEqual(files(got), want) {
		t.Errorf("order %v, want %v", files(got), want)
	}
	if got[0].Title != "New" {
		t.Errorf("title %q", got[0].Title)
	}
	if none, err := find(filepath.Join(d, "missing"), ""); err != nil || none != nil {
		t.Errorf("a missing folder is no notes: %v %v", none, err)
	}
}

func TestSearch(t *testing.T) {
	d := t.TempDir()
	note(t, d, "shopping.md", "# Shopping\n\nmilk\nBread and butter\n", 1)
	note(t, d, "bread.md", "# Bread recipe\n\nflour, water\n", 2)
	note(t, d, "trip.md", "# Trip\n\npack the bread knife\n", 3)
	note(t, d, "other.md", "# Other\n\nnothing here\n", 4)

	got, err := find(d, "bread")
	if err != nil {
		t.Fatal(err)
	}
	// the title match first, then the text matches, newest first
	if want := []string{"bread.md", "trip.md", "shopping.md"}; !reflect.DeepEqual(files(got), want) {
		t.Fatalf("got %v, want %v", files(got), want)
	}
	if got[0].Line != "" || got[1].Line != "pack the bread knife" || got[2].Line != "Bread and butter" {
		t.Errorf("lines %q %q %q", got[0].Line, got[1].Line, got[2].Line)
	}
	// every word, each in the title or the text, any case
	got, _ = find(d, "SHOPPING Milk")
	if want := []string{"shopping.md"}; !reflect.DeepEqual(files(got), want) {
		t.Errorf("two words: %v", files(got))
	}
	got, _ = find(d, "bread zebra")
	if len(got) != 0 {
		t.Errorf("a word nowhere matches nothing: %v", files(got))
	}
}

func TestMatch(t *testing.T) {
	for _, c := range []struct {
		title, text string
		words       []string
		ok, inTitle bool
		line        string
	}{
		{"Groceries", "# Groceries\neggs", []string{"groc"}, true, true, ""},
		{"Groceries", "# Groceries\n  Eggs, milk  \n", []string{"groc", "milk"}, true, false, "Eggs, milk"},
		{"Groceries", "# Groceries\neggs", []string{"tea"}, false, false, ""},
		{"Title", "no newline at the end", []string{"end"}, true, false, "no newline at the end"},
	} {
		ok, inTitle, line := Match(c.title, c.text, c.words)
		if ok != c.ok || inTitle != c.inTitle || line != c.line {
			t.Errorf("Match(%q, %v) = %v %v %q", c.title, c.words, ok, inTitle, line)
		}
	}
}

func TestCreate(t *testing.T) {
	d := filepath.Join(t.TempDir(), "notes")
	p, err := create(d, "  Call the bank: re/mortgage ")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "call-the-bank-re-mortgage.md" {
		t.Errorf("file %s", filepath.Base(p))
	}
	data, _ := os.ReadFile(p)
	if string(data) != "# Call the bank: re/mortgage\n\n" {
		t.Errorf("body %q", data)
	}
	// the same title again is a second file, not the first written over
	p2, err := create(d, "Call the bank: re/mortgage")
	if err != nil || filepath.Base(p2) != "call-the-bank-re-mortgage-2.md" {
		t.Errorf("second: %s %v", p2, err)
	}
	if data, _ := os.ReadFile(p); string(data) != "# Call the bank: re/mortgage\n\n" {
		t.Errorf("the first was changed: %q", data)
	}
	p3, err := create(d, "")
	if err != nil || filepath.Base(p3) != "untitled.md" {
		t.Errorf("untitled: %s %v", p3, err)
	}
}

func TestRemove(t *testing.T) {
	d := t.TempDir()
	note(t, d, "gone.md", "# Gone\n", 1)
	dst, err := remove(d, "gone.md")
	if err != nil {
		t.Fatal(err)
	}
	if dst != filepath.Join(d, deletedDir, "gone.md") {
		t.Errorf("moved to %s", dst)
	}
	if _, err := os.Stat(filepath.Join(d, "gone.md")); !os.IsNotExist(err) {
		t.Errorf("still in the folder: %v", err)
	}
	if got, _ := find(d, ""); len(got) != 0 {
		t.Errorf("a deleted note is not listed: %v", files(got))
	}
	// a second note of the same name does not replace the first deleted
	note(t, d, "gone.md", "# Gone again\n", 2)
	dst, err = remove(d, "gone.md")
	if err != nil || dst != filepath.Join(d, deletedDir, "gone-2.md") {
		t.Errorf("second delete: %s %v", dst, err)
	}
	if data, _ := os.ReadFile(filepath.Join(d, deletedDir, "gone.md")); string(data) != "# Gone\n" {
		t.Errorf("the first deleted note was changed: %q", data)
	}
	for _, bad := range []string{"missing.md", "../escape.md", "sub/x.md", "notes.txt", ""} {
		if _, err := remove(d, bad); err == nil {
			t.Errorf("remove(%q) worked", bad)
		}
	}
}

func TestNewTitle(t *testing.T) {
	for _, c := range []struct {
		id, want string
		ok       bool
	}{
		{"+new", "", true},
		{"+new/a/b c", "a/b c", true},
		{"+newer.md", "", false},
		{"groceries.md", "", false},
	} {
		got, ok := newTitle(c.id)
		if got != c.want || ok != c.ok {
			t.Errorf("newTitle(%q) = %q %v", c.id, got, ok)
		}
	}
}
