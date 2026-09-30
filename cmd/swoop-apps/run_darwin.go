//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// run performs action on the application at path. "" and "open" launch
// it with the system `open` tool, which returns as soon as Launch
// Services has accepted the request; "reveal" shows it in Finder;
// "copy-path" puts the path on the clipboard. `open`'s stderr is passed
// through so a bad path shows a real message.
func run(path, action string) error {
	switch action {
	case "", "open":
		cmd := exec.Command("open", path)
		cmd.Stderr = stderr()
		return cmd.Run()
	case "reveal":
		cmd := exec.Command("open", "-R", path)
		cmd.Stderr = stderr()
		return cmd.Run()
	case "copy-path":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(path)
		return cmd.Run()
	}
	return fmt.Errorf("no action %q for an app", action)
}
