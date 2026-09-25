package main

import (
	"os"
	"os/exec"
	"syscall"
)

// startWorker runs `swoop-ai work <id>` detached: a new process group
// with no console, so the Enter that sent the prompt returns at once.
func startWorker(id string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "work", id)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000} // CREATE_NO_WINDOW
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
