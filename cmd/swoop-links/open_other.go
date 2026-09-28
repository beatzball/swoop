//go:build !darwin && !linux

package main

import "errors"

func openLink(string, string) error { return errors.New("not implemented on this OS yet") }

func copyText(string) error { return errors.New("not implemented on this OS yet") }
