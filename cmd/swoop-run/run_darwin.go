//go:build darwin

package main

import "os/exec"

// run opens the application at path with the system `open` tool. `open`
// returns as soon as Launch Services has accepted the request, so this is
// fast and does not wait for the app to finish starting. Its stderr is
// passed through so a bad path shows a real message.
func run(path string) error {
	cmd := exec.Command("open", path)
	cmd.Stderr = stderr()
	return cmd.Run()
}
