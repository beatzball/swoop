//go:build !darwin

package main

// managedByLaunchd is a macOS question; elsewhere nothing manages the
// watcher but the launcher.
func managedByLaunchd() bool { return false }
