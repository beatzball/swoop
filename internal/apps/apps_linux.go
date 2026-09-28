//go:build linux

package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/beatzball/swoop/internal/protocol"
)

// RevealTitle is the menu label for "reveal": the file manager differs by
// desktop, so it is not named.
const RevealTitle = "Show in folder"

func list() ([]protocol.Item, error) {
	return desktopItems(dataDirs()), nil
}

// dataDirs are the applications directories in XDG precedence order: the
// user's own first, then each of XDG_DATA_DIRS. ~/.local/share is listed
// even when XDG_DATA_HOME points elsewhere, because apps installed by
// tools that ignore the variable still land there. A directory named twice
// is read once.
func dataDirs() []string {
	home, _ := os.UserHomeDir()
	bases := []string{}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		bases = append(bases, v)
	}
	bases = append(bases, filepath.Join(home, ".local", "share"))
	system := os.Getenv("XDG_DATA_DIRS")
	if system == "" {
		system = "/usr/local/share:/usr/share"
	}
	bases = append(bases, strings.Split(system, ":")...)

	seen := map[string]bool{}
	var dirs []string
	for _, b := range bases {
		if b == "" {
			continue
		}
		d := filepath.Join(filepath.Clean(b), "applications")
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	return dirs
}
