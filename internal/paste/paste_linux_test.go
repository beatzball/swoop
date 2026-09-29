//go:build linux

package paste

import (
	"os"
	"path/filepath"
	"testing"
)

// Missing speaks for the same tools Copy uses: with neither on PATH it
// names both, and with either one it is quiet.
func TestMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if Missing() == "" {
		t.Fatal("no wl-copy, no xclip, and Missing says nothing")
	}
	for _, name := range []string{"wl-copy", "xclip"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\ncat >/dev/null\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir)
		if got := Missing(); got != "" {
			t.Errorf("with %s: %q", name, got)
		}
	}
}
