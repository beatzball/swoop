//go:build linux

package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// openLink hands the link to xdg-open, or to the named app as its first
// argument, the way a browser or an editor takes one. The app runs in a
// session of its own: swoop-run replaced fzf, and the launcher's terminal
// closes once it exits, so a child that shared its session would be
// hung up with it.
func openLink(link, app string) error {
	name, args := "xdg-open", []string{link}
	if app != "" {
		name = app
	}
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stderr = os.Stderr
	if app == "" {
		return cmd.Run()
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// copyText puts s on the clipboard with wl-copy on Wayland or xclip on X,
// whichever is installed.
func copyText(s string) error {
	for _, c := range [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(s)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New("no clipboard tool: install wl-clipboard or xclip")
}
