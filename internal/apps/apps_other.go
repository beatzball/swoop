//go:build !darwin

package apps

import "github.com/beatzball/swoop/internal/protocol"

// list is the placeholder for Linux (.desktop files) and Windows (Start
// Menu shortcuts): no apps yet, and no error, so the launcher still opens
// with the extensions' rows. An error here ended bin/swoop before fzf
// started, which the end-to-end test found on its first Linux run.
func list() ([]protocol.Item, error) {
	return nil, nil
}
