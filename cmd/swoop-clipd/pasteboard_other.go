//go:build !darwin || !cgo

package main

import "errors"

type pasteboard struct{}

func openPasteboard() (pasteboard, error) {
	return pasteboard{}, errors.New("no pasteboard source on this OS yet")
}

func (pasteboard) ChangeCount() int64   { return 0 }
func (pasteboard) Concealed() bool      { return true }
func (pasteboard) Text() (string, bool) { return "", false }
func (pasteboard) Types() string        { return "" }
func (pasteboard) Frontmost() string    { return "" }
