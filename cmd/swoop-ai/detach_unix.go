//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// startWorker runs `swoop-ai work <id>` in its own session with no
// terminal and no output, the way swoop-clipd start does, so the Enter
// that sent the prompt returns at once and the worker outlives the
// launcher if it must. It inherits the environment, which is how it
// learns fzf's socket and key.
func startWorker(id string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "work", id)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
