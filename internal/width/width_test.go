package width

import "testing"

func TestString(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"\x1b[1mbold\x1b[22m", 4},
		{"\x1b[38;5;215mcode\x1b[39m plain", 10},
		{"\x1b]8;;http://x\x07link\x1b]8;;\x07", 4},
		{"日本語", 6},
		{"a日b", 4},
		{"café", 4},
		{"café", 4},
		{"🙂 ok", 5},
		{"👍🏽", 4}, // two wide runes, as most terminals draw the pair
		{"tab\there", 7},
		{"─────", 5},
	}
	for _, c := range cases {
		if got := String(c.in); got != c.want {
			t.Errorf("String(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestStrip(t *testing.T) {
	if got := Strip("\x1b[1mhi\x1b[22m there \x1b[38;5;110m│\x1b[39m"); got != "hi there │" {
		t.Fatalf("got %q", got)
	}
	if got := Strip("plain"); got != "plain" {
		t.Fatalf("plain untouched: %q", got)
	}
	if got := Strip("cut \x1b[3"); got != "cut " {
		t.Fatalf("an unfinished sequence is dropped, not shown: %q", got)
	}
}
