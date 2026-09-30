package main

import (
	"testing"

	"github.com/beatzball/swoop/internal/protocol"
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
