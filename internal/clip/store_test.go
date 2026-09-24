package clip

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func temp(t *testing.T) Store {
	t.Helper()
	return Store{Path: filepath.Join(t.TempDir(), "sub", "history.jsonl")}
}

func TestAppendAndAllNewestFirst(t *testing.T) {
	s := temp(t)
	base := time.Unix(1000, 0)
	for i, text := range []string{"one", "two", "three"} {
		if err := s.Append(text, base.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.All()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range all {
		got = append(got, e.Text)
	}
	if strings.Join(got, ",") != "three,two,one" {
		t.Fatalf("want newest first, got %v", got)
	}
}

func TestAppendSkipsEmptyAndRepeats(t *testing.T) {
	s := temp(t)
	now := time.Unix(1000, 0)
	_ = s.Append("  \n", now)
	_ = s.Append("same", now)
	_ = s.Append("same", now.Add(time.Second))
	_ = s.Append("other", now.Add(2*time.Second))
	_ = s.Append("same", now.Add(3*time.Second))
	all, _ := s.All()
	if len(all) != 3 {
		t.Fatalf("want 3 entries (same, other, same), got %d: %+v", len(all), all)
	}
}

func TestAppendCutsLongText(t *testing.T) {
	s := temp(t)
	if err := s.Append(strings.Repeat("x", MaxText+10), time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	all, _ := s.All()
	if len(all[0].Text) != MaxText {
		t.Fatalf("want %d bytes kept, got %d", MaxText, len(all[0].Text))
	}
}

func TestFindAndGet(t *testing.T) {
	s := temp(t)
	now := time.Unix(1000, 0)
	_ = s.Append("Hello World", now)
	_ = s.Append("goodbye", now.Add(time.Second))
	found, _ := s.Find("WORLD")
	if len(found) != 1 || found[0].Text != "Hello World" {
		t.Fatalf("Find is case-insensitive substring: %+v", found)
	}
	e, err := s.Get(strconv.FormatInt(found[0].ID, 10))
	if err != nil || e.Text != "Hello World" {
		t.Fatalf("Get by id: %+v %v", e, err)
	}
	if _, err := s.Get("nope"); err == nil {
		t.Fatal("bad id accepted")
	}
	if _, err := s.Get("42"); err == nil {
		t.Fatal("unknown id accepted")
	}
}

func TestCompactKeepsNewest(t *testing.T) {
	s := temp(t)
	base := time.Unix(1000, 0)
	for i := 0; i < 2*MaxEntries+1; i++ {
		if err := s.Append("entry "+strconv.Itoa(i), base.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := s.All()
	if len(all) != MaxEntries {
		t.Fatalf("want %d after compaction, got %d", MaxEntries, len(all))
	}
	if all[0].Text != "entry "+strconv.Itoa(2*MaxEntries) {
		t.Fatalf("newest lost: %q", all[0].Text)
	}
}

func TestMissingFileIsEmpty(t *testing.T) {
	s := temp(t)
	all, err := s.All()
	if err != nil || len(all) != 0 {
		t.Fatalf("missing file: %v %v", all, err)
	}
}

func TestTitle(t *testing.T) {
	cases := map[string]string{
		"hello":                     "hello",
		"\n\n  first   line \nmore": "first line",
		strings.Repeat("a", 70):     strings.Repeat("a", 59) + "…",
		"   ":                       "",
	}
	for in, want := range cases {
		if got := Title(in, 60); got != want {
			t.Errorf("Title(%q) = %q, want %q", in, got, want)
		}
	}
}
