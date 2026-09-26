package main

import (
	"os"
	"path/filepath"
)

// managedByLaunchd says whether make install has written the watcher's
// launchd agent. The plist is the sign; a running service is not
// checked, because the whole point is the moment when it is not running
// yet.
func managedByLaunchd() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, "Library", "LaunchAgents", "dev.swoop.clipd.plist"))
	return err == nil
}
