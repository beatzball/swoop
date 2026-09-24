package protocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	in := Item{ID: "/Applications/Safari.app", Kind: "app", Icon: "", Title: "Safari", Subtitle: "/Applications"}
	out, err := Parse(in.Line() + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round trip changed the item:\n in  %+v\n out %+v", in, out)
	}
}

func TestLineCleansSeparators(t *testing.T) {
	it := Item{ID: "x", Kind: "text", Title: "a\tb\nc", Subtitle: "d\re"}
	line := it.Line()
	if strings.Count(line, "\t") != Fields-1 {
		t.Fatalf("a tab inside a field leaked into the line: %q", line)
	}
	if strings.ContainsAny(line, "\n\r") {
		t.Fatalf("a newline inside a field leaked into the line: %q", line)
	}
}

func TestParseRejectsBadLines(t *testing.T) {
	for _, bad := range []string{"", "only\tfour\tfields\there", "\tapp\t\tNo id\t"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted a bad line", bad)
		}
	}
}

func TestWrite(t *testing.T) {
	var buf bytes.Buffer
	items := []Item{{ID: "1", Kind: "app", Title: "One"}, {ID: "2", Kind: "app", Title: "Two"}}
	if err := Write(&buf, items); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != len(items) {
		t.Fatalf("want %d lines, got %d: %q", len(items), len(lines), buf.String())
	}
	for i, l := range lines {
		it, err := Parse(l)
		if err != nil {
			t.Fatal(err)
		}
		if it != items[i] {
			t.Fatalf("line %d: want %+v, got %+v", i, items[i], it)
		}
	}
}
