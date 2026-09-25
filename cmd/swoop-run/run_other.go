//go:build !darwin

package main

import "errors"

func run(string, string) error {
	return errors.New("not implemented on this OS yet")
}
