//go:build !darwin || !cgo

package main

import "errors"

func define(string) (string, error) {
	return "", errors.New("no dictionary source on this OS yet")
}
