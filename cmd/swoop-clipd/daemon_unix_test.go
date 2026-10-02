//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/beatzball/swoop/internal/clip"
)

// The lock file names the watcher that holds it, and only while it is
// held: `pid` is how a script stops one tool's watcher and leaves the
// watcher of another tool running.
func TestHolderIsTheWatcherThatSigned(t *testing.T) {
	store := clip.Store{Path: filepath.Join(t.TempDir(), "clipboard", "history.jsonl")}
	if pid, ok := holder(store); ok {
		t.Fatalf("no watcher runs, and the holder is %d", pid)
	}

	unlock, err := lock(store)
	if err != nil {
		t.Fatal(err)
	}
	// Held, by a watcher from before the file held an id: not known.
	if pid, ok := holder(store); ok {
		t.Fatalf("a lock nobody signed has the holder %d", pid)
	}
	if err := sign(store); err != nil {
		t.Fatal(err)
	}
	if pid, ok := holder(store); !ok || pid != os.Getpid() {
		t.Fatalf("holder = %d, %v; want this process, %d", pid, ok, os.Getpid())
	}

	// Let go: the id still in the file is nobody's now.
	unlock()
	if pid, ok := holder(store); ok {
		t.Fatalf("the lock is free, and the holder is %d", pid)
	}
}

// Another tool's folder is another lock: a watcher there is not this
// tool's.
func TestHolderIsPerFolder(t *testing.T) {
	mine := clip.Store{Path: filepath.Join(t.TempDir(), "clipboard", "history.jsonl")}
	theirs := clip.Store{Path: filepath.Join(t.TempDir(), "clipboard", "history.jsonl")}
	unlock, err := lock(theirs)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if err := sign(theirs); err != nil {
		t.Fatal(err)
	}
	if pid, ok := holder(mine); ok {
		t.Fatalf("another tool's watcher is the holder here: %d", pid)
	}
}
