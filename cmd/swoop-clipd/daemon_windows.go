//go:build windows

package main

import (
	"errors"

	"github.com/beatzball/swoop/internal/clip"
)

func lock(clip.Store) (func(), error) {
	return nil, errors.New("the watcher is not implemented on Windows yet")
}

func running(clip.Store) bool { return false }

func start(clip.Store) error { return nil }
