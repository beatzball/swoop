package main

import "os"

// alive says whether the process is still there. FindProcess succeeds for
// any pid on Windows, so this is only ever a yes; the worker's own five
// minute limit still ends a wait that would otherwise be forever.
func alive(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}
