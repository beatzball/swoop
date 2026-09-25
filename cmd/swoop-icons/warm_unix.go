//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/beatzball/swoop/internal/bundle"
)

// startWarmer runs `swoop-icons -warm` on apps in its own session with no
// terminal and no output, the way swoop-clipd start does, so it outlives
// this run and fzf never waits on it. It returns at once.
func startWarmer(apps []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, append([]string{"-warm", "--"}, apps...)...)
	// Stdin, Stdout and Stderr left nil are /dev/null.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Not Wait: the child is on its own now. Release it so it is not a
	// zombie of a parent that has already exited.
	return cmd.Process.Release()
}

// warm converts the icon of each app into the cache. It holds an
// exclusive flock on a file in the cache directory while it works, and
// gives up at once if another warmer holds it: two launches in a row on a
// cold cache would otherwise convert every icon twice, side by side. The
// lock goes with the process, so it never goes stale.
func warm(apps []string) error {
	dir := bundle.IconCacheDir()
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "warm.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil
	}
	for _, app := range apps {
		// An icon that will not convert keeps its glyph; the rest still
		// get theirs.
		_, _ = bundle.Describe(app).IconPNG(iconPx)
	}
	return nil
}
