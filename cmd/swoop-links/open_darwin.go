//go:build darwin

package main

import (
	"os"
	"os/exec"
	"strings"
)

// openLink hands the link to `open`, which knows URLs, paths, and any
// scheme an installed app claims; with an app named, `open -a`, which
// takes a name or a bundle path. stderr passes through so a link nothing
// can open shows a real message.
func openLink(link, app string) error {
	args := []string{link}
	if app != "" {
		args = []string{"-a", app, link}
	}
	cmd := exec.Command("open", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyText(s string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}
