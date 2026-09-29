package main

import (
	"testing"

	"github.com/beatzball/swoop/internal/protocol"
)

func TestMatchingNeedsEveryWordInTheTitle(t *testing.T) {
	items := []protocol.Item{{Title: "Left Half"}, {Title: "Left Third"}, {Title: "Right Half"}}
	cases := map[string]int{"": 3, "left": 2, "HALF left": 1, "  half  ": 2, "top": 0}
	for q, want := range cases {
		if got := len(matching(items, q)); got != want {
			t.Errorf("matching(%q) kept %d, want %d", q, got, want)
		}
	}
}
