//go:build darwin

package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/beatzball/swoop/internal/protocol"
)

// dirs are the places macOS keeps launchable applications. Each is read one
// level deep and never recursed: an app bundle holds helper .app bundles
// inside it, and those must not show up as results. Utilities is listed on
// its own for the same reason. "~" is the user's home.
var dirs = []string{
	"/Applications",
	"/Applications/Utilities",
	"/System/Applications",
	"/System/Applications/Utilities",
	"~/Applications",
}

func list() ([]protocol.Item, error) {
	home, _ := os.UserHomeDir()
	var items []protocol.Item
	for _, shown := range dirs {
		// dir is the real path, for reading and for the id.
		dir := shown
		if strings.HasPrefix(dir, "~") {
			dir = home + dir[1:]
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			// A missing directory is normal: ~/Applications often does not
			// exist. Anything else (permissions) is also not worth failing
			// the whole list for; the other directories still count.
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".app") {
				continue
			}
			items = append(items, protocol.Item{
				// The bundle path is the id: stable, unique, and exactly
				// what `open` needs. swoop-run gets it back untouched.
				ID:    filepath.Join(dir, name),
				Kind:  Kind,
				Icon:  Icon,
				Title: strings.TrimSuffix(name, ".app"),
				// No subtitle: the folder is in the preview, where it is
				// read once, not on every row, where it was noise.
			})
		}
	}
	return items, nil
}
