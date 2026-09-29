//go:build !darwin && !linux && !windows

package paste

import "errors"

const pasteKey = "ctrl+V"

func Copy(string) error { return errors.New("not implemented on this OS yet") }

func Missing() string { return "no clipboard on this OS yet" }

func Clipboard() (string, error) { return "", errors.New("not implemented on this OS yet") }

func notify(string) error { return nil }
