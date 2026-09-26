//go:build !windows

package main

import (
	"errors"
	"syscall"
)

// alive says whether the process is still there. Signal 0 is delivered to
// nobody, but fails with ESRCH when there is no such process; EPERM means
// there is one, just not ours.
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
