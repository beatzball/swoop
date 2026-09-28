//go:build !darwin && !linux

package apps

import "github.com/beatzball/swoop/internal/protocol"

// RevealTitle is the menu label for "reveal".
const RevealTitle = "Show in folder"

// list is the placeholder for Windows (Start Menu shortcuts) and anything
// else: no apps yet, and no error, so the launcher still opens with the
// extensions' rows. An error here ended bin/swoop before fzf started, which
// the end-to-end test found on its first Linux run.
func list() ([]protocol.Item, error) {
	return nil, nil
}
