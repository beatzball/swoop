//go:build linux

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/beatzball/swoop/internal/apps"
)

// run performs action on the application whose .desktop file is at path.
// "" and "open" hand the file to `gio launch`, which knows every rule of
// the spec (Terminal=true, DBusActivatable, startup notification); without
// gio, the Exec line is run by hand. "reveal" opens the file's folder in
// the file manager; "copy-path" puts the path on the clipboard.
func run(path, action string) error {
	switch action {
	case "", "open":
		if gio, err := exec.LookPath("gio"); err == nil {
			cmd := exec.Command(gio, "launch", path)
			cmd.Stderr = stderr()
			return cmd.Run()
		}
		return launchExec(path)
	case "reveal":
		cmd := exec.Command("xdg-open", filepath.Dir(path))
		cmd.Stderr = stderr()
		return cmd.Run()
	case "copy-path":
		return copyText(path)
	}
	return fmt.Errorf("no action %q for an app", action)
}

// launchExec runs the entry's Exec line through sh, in a session of its
// own. swoop-run, which ran this tool, replaced fzf through become, and
// the launcher's terminal closes once it exits; Setsid keeps the app from
// being hung up with it. sh reads the quoting the spec allows in Exec, so
// it is not parsed here.
func launchExec(path string) error {
	d, err := apps.ReadDesktop(path)
	if err != nil {
		return err
	}
	line := apps.StripFieldCodes(d.Exec)
	if line == "" {
		return fmt.Errorf("%s has no Exec line", path)
	}
	cmd := exec.Command("sh", "-c", line)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stderr = stderr()
	if err := cmd.Start(); err != nil {
		return err
	}
	// Not waited for: the app runs on after this tool exits. Releasing
	// it lets the Go runtime forget the child without a zombie wait.
	return cmd.Process.Release()
}

// copyText puts s on the clipboard with wl-copy on Wayland or xclip on X,
// whichever is installed. Neither ships with every desktop, so their
// absence is an error the user can act on rather than a silent no-op.
func copyText(s string) error {
	for _, c := range [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(s)
		cmd.Stderr = stderr()
		return cmd.Run()
	}
	return errors.New("no clipboard tool: install wl-clipboard or xclip")
}
