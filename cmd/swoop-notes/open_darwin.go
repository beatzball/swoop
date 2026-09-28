//go:build darwin

package main

import (
	"os"
	"os/exec"
)

// openFile hands the note to `open`, which opens it in the app that owns
// .md files. stderr passes through so a real message shows when none
// does.
func openFile(p string) error {
	cmd := exec.Command("open", p)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// reveal shows the file selected in Finder.
func reveal(p string) error {
	cmd := exec.Command("open", "-R", p)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
