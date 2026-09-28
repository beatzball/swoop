//go:build !darwin || !cgo

package main

import "errors"

// Linux (X11 through wmctrl, Wayland per compositor) and Windows
// (SetWindowPos) come with their frames.
var errNotHere = errors.New("not implemented on this OS yet")

type Window struct {
	App, Title string
	Frame      Rect
}

func trusted() bool { return true }

func askTrust() error { return errNotHere }

func focused() (Window, error) { return Window{}, errNotHere }

func displays() ([]Display, error) { return nil, errNotHere }

func move(Window, Rect) error { return errNotHere }
