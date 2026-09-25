package picture

import (
	"os"
	"strings"
)

// Enabled says whether to send pictures at all. A terminal that does not
// speak the protocol shows a transmitted picture as a pane of noise and a
// placeholder cell as a box, so the tools that send them ask this first
// and fall back to glyphs and text when the answer is no.
//
// The terminal is not asked: a one-shot process would pay a round trip on
// every run, and fzf's preview command has no time for that. The
// environment says what is known. SWOOP_PICTURES=1 or 0 wins when set;
// otherwise the terminals that draw them are recognized by the variables
// they set for their children. libghostty, in the frame, sets
// TERM_PROGRAM=ghostty like Ghostty.app does, so the frame needs no
// setting. tmux is deliberately not on the list: it passes the escapes
// through only when told to, and swoop does not run under it.
func Enabled() bool {
	return enabled(os.Getenv)
}

func enabled(getenv func(string) string) bool {
	switch getenv("SWOOP_PICTURES") {
	case "1":
		return true
	case "0":
		return false
	}
	switch getenv("TERM_PROGRAM") {
	case "ghostty", "WezTerm":
		return true
	}
	if getenv("KITTY_WINDOW_ID") != "" || getenv("KONSOLE_VERSION") != "" {
		return true
	}
	term := getenv("TERM")
	return strings.HasPrefix(term, "xterm-kitty") || strings.HasPrefix(term, "xterm-ghostty")
}
