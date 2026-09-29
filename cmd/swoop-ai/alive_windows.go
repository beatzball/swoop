package main

import (
	"errors"
	"syscall"
)

const (
	// processQueryLimitedInformation is the least access that lets
	// GetExitCodeProcess read a process: PROCESS_QUERY_LIMITED_INFORMATION.
	processQueryLimitedInformation = 0x1000
	// stillActive is the exit code GetExitCodeProcess reports for a
	// process that has not exited: STILL_ACTIVE.
	stillActive = 259
)

// alive says whether the process is still there. os.FindProcess cannot
// tell: it succeeds for any pid on Windows. OpenProcess fails for a pid
// with no process behind it, and for one that has exited GetExitCodeProcess
// gives its exit code instead of STILL_ACTIVE. Access denied means there is
// a process, just not ours, as EPERM does on Unix.
//
// A process that exits with 259 of its own reads as alive; the worker's
// five minute limit still ends that wait.
func alive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		// Opened but unreadable: there is a process, so do not call it dead.
		return true
	}
	return code == stillActive
}
