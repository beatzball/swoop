//go:build !darwin && !linux

package paste

import "errors"

const pasteKey = "ctrl+V"

func Copy(string) error { return errors.New("not implemented on this OS yet") }

func keystroke(string) (string, error) { return "", errors.New("not implemented on this OS yet") }

func notify(string) error { return nil }
