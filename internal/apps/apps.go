// Package apps finds the applications a user can launch. It is the first
// built-in source and speaks the same contract an extension would: a list of
// protocol items. The per-OS part lives behind build tags in the files next
// to this one, so this file has no idea what an ".app" is.
package apps

import (
	"sort"
	"strings"

	"github.com/beatzball/swoop/internal/protocol"
)

// Kind is the protocol kind every item from this source carries.
const Kind = "app"

// Icon is the glyph shown on every app row until per-app icons exist. It is
// a Nerd Font codepoint (nf-fa-rocket), which libghostty draws from its
// bundled fallback font with nothing installed. See the "Design: icons,
// emoji, and images" issue for why rows get a glyph and not a picture.
const Icon = ""

// List returns every launchable application, sorted by title without regard
// to case, so "iTerm" does not sink below "Zoom".
func List() ([]protocol.Item, error) {
	items, err := list()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
	return items, nil
}
