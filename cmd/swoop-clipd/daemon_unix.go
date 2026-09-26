//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/beatzball/swoop/internal/clip"
)

// lockPath is next to the history file. The watcher holds an exclusive
// flock on it for as long as it runs, which is how "is one running" is
// answered without pid files that go stale.
func lockPath(store clip.Store) string {
	return filepath.Join(filepath.Dir(store.Path), "watcher.lock")
}

// lock takes the watcher's lock, or says another watcher has it.
func lock(store clip.Store) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(store.Path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath(store), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("another watcher is running")
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}

// running reports whether a watcher holds the lock.
func running(store clip.Store) bool {
	unlock, err := lock(store)
	if err != nil {
		return true
	}
	unlock()
	return false
}

// start launches `swoop-clipd run` in its own session with no terminal, so
// it outlives the launcher, and returns at once. If a watcher already
// holds the lock there is nothing to do. Its output goes to a log next to
// the history file.
//
// When launchd owns the watcher (make install wrote its agent) this does
// nothing either, whether or not the lock is held: launchd's KeepAlive
// brings its watcher back, and a watcher started here in the second
// before it did would hold the lock instead, and keep launchd's out for
// good. That happened three times.
func start(store clip.Store) error {
	if managedByLaunchd() {
		return nil
	}
	if running(store) {
		return nil
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(filepath.Dir(store.Path), "watcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(self, "run")
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}
	// Not Wait: the child is on its own now. Release it so it is not a
	// zombie of a parent that has already exited.
	return cmd.Process.Release()
}
