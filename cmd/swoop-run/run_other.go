//go:build !darwin

package main

import "errors"

func run(string) error {
	return errors.New("not implemented on this OS yet")
}
