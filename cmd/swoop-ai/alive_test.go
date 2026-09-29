package main

import (
	"os"
	"os/exec"
	"testing"
)

// alive has one file per OS; this runs against whichever is built. A
// worker that has exited must read as dead, or the pane shows dots until
// the five minute limit instead of saying the worker stopped.
func TestAlive(t *testing.T) {
	if !alive(os.Getpid()) {
		t.Fatal("this process reads as dead")
	}
	// The test binary itself, told to run no tests: it exits at once.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if alive(cmd.Process.Pid) {
		t.Fatalf("pid %d exited and was waited for, and reads as alive", cmd.Process.Pid)
	}
}
