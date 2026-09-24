//go:build !darwin

package apps

import (
	"errors"

	"github.com/beatzball/swoop/internal/protocol"
)

// list is the placeholder for Linux (.desktop files) and Windows (Start Menu
// shortcuts). The build still succeeds on those OSes so the shared code and
// CI can run there; only this source says "not yet".
func list() ([]protocol.Item, error) {
	return nil, errors.New("apps: not implemented on this OS yet")
}
