//go:build !darwin

// Package bundle reads what an application says about itself. Only macOS
// bundles are understood so far; elsewhere the name is the file name and
// there is no icon.
package bundle

import (
	"errors"
	"path/filepath"
)

// Info is the subset of an application's metadata swoop shows.
type Info struct {
	Name     string
	Version  string
	BundleID string
}

// Describe returns what can be known without reading anything.
func Describe(path string) Info {
	return Info{Name: filepath.Base(path)}
}

// ErrNoIcon says there is no icon to show.
var ErrNoIcon = errors.New("bundle: icons not implemented on this OS yet")

// IconPNG always reports ErrNoIcon here.
func (Info) IconPNG(int) ([]byte, error) { return nil, ErrNoIcon }

// ErrNotCached says an icon has not been converted yet. Nothing is ever
// converted here, so it is never returned.
var ErrNotCached = errors.New("bundle: icon not cached yet")

// CachedIconPNG always reports ErrNoIcon here.
func (Info) CachedIconPNG(int) ([]byte, error) { return nil, ErrNoIcon }

// IconCacheDir is "": there is nothing to cache.
func IconCacheDir() string { return "" }
