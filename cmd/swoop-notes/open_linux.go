//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// openFile hands the note to xdg-open, in a session of its own:
// swoop-run replaced fzf, and the launcher's terminal closes once it
// exits, so a child that shared its session would be hung up with it.
func openFile(p string) error {
	return xdgOpen(p)
}

// reveal opens the note's folder. xdg-open has no way to select a file
// in it.
func reveal(p string) error {
	return xdgOpen(filepath.Dir(p))
}

func xdgOpen(target string) error {
	cmd := exec.Command("xdg-open", target)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
