//go:build !darwin && !linux

package main

import "errors"

func openFile(string) error { return errors.New("not implemented on this OS yet") }

func reveal(string) error { return errors.New("not implemented on this OS yet") }
