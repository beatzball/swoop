//go:build darwin

// Package bundle reads what a macOS application bundle says about itself:
// its name, version, identifier, and icon. It is imported by swoop-preview
// only. swoop-list must stay lean, so the plist and icns decoders live here
// and not in the apps package.
package bundle

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackmordaunt/icns/v3"
	"howett.net/plist"
)

// Info is the subset of Info.plist swoop shows.
type Info struct {
	Name     string
	Version  string
	BundleID string
	iconFile string // CFBundleIconFile, possibly without its .icns suffix
	path     string
}

// Describe reads the bundle at path. A bundle with no readable Info.plist
// still gets a Name, taken from the directory, so the preview never comes
// up blank for a launchable app.
func Describe(path string) Info {
	info := Info{path: path, Name: strings.TrimSuffix(filepath.Base(path), ".app")}
	data, err := os.ReadFile(filepath.Join(path, "Contents", "Info.plist"))
	if err != nil {
		return info
	}
	var p struct {
		DisplayName string `plist:"CFBundleDisplayName"`
		Name        string `plist:"CFBundleName"`
		Version     string `plist:"CFBundleShortVersionString"`
		BundleID    string `plist:"CFBundleIdentifier"`
		IconFile    string `plist:"CFBundleIconFile"`
	}
	if _, err := plist.Unmarshal(data, &p); err != nil {
		return info
	}
	if p.DisplayName != "" {
		info.Name = p.DisplayName
	} else if p.Name != "" {
		info.Name = p.Name
	}
	info.Version = p.Version
	info.BundleID = p.BundleID
	info.iconFile = p.IconFile
	return info
}

// ErrNoIcon says the bundle keeps its icon somewhere swoop cannot read yet:
// apps built with an asset catalog have no .icns file, only Assets.car.
var ErrNoIcon = errors.New("bundle: no .icns icon")

// maxIconPx is the largest icon size worth sending to a terminal. Icons
// ship at up to 1024px; a preview pane is a few hundred pixels across, and
// every extra byte is base64 through a pipe on every cursor move.
const maxIconPx = 256

// IconPNG returns the bundle's icon as PNG bytes, converting the .icns once
// and keeping the result in the user's cache directory. The cache key is
// the icns path plus its size and modification time, so an app update
// invalidates it on its own. A cache that cannot be written is not an
// error: the conversion just runs again next time.
func (info Info) IconPNG() ([]byte, error) {
	icnsPath, err := info.icnsPath()
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(icnsPath)
	if err != nil {
		return nil, ErrNoIcon
	}
	key := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", icnsPath, st.Size(), st.ModTime().UnixNano())))
	cachePath := ""
	if dir, err := os.UserCacheDir(); err == nil {
		cachePath = filepath.Join(dir, "swoop", "icons", hex.EncodeToString(key[:])+".png")
		if data, err := os.ReadFile(cachePath); err == nil {
			return data, nil
		}
	}

	data, err := convert(icnsPath)
	if err != nil {
		return nil, err
	}
	if cachePath != "" {
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
			// Write to a temp name and rename, so a preview that runs while
			// another is still writing never reads half a file.
			tmp := cachePath + ".tmp"
			if err := os.WriteFile(tmp, data, 0o644); err == nil {
				_ = os.Rename(tmp, cachePath)
			}
		}
	}
	return data, nil
}

func (info Info) icnsPath() (string, error) {
	res := filepath.Join(info.path, "Contents", "Resources")
	if info.iconFile != "" {
		name := info.iconFile
		if !strings.HasSuffix(name, ".icns") {
			name += ".icns"
		}
		return filepath.Join(res, name), nil
	}
	// No CFBundleIconFile. Some bundles still ship an .icns; take the first.
	matches, _ := filepath.Glob(filepath.Join(res, "*.icns"))
	if len(matches) == 0 {
		return "", ErrNoIcon
	}
	return matches[0], nil
}

// convert decodes every size in the .icns and keeps the largest one that is
// not bigger than maxIconPx, falling back to the smallest available.
func convert(icnsPath string) ([]byte, error) {
	f, err := os.Open(icnsPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	images, err := icns.DecodeAll(f)
	if err != nil || len(images) == 0 {
		return nil, fmt.Errorf("bundle: decode %s: %w", filepath.Base(icnsPath), err)
	}
	var best image.Image
	for _, img := range images {
		w := img.Bounds().Dx()
		if best == nil {
			best = img
			continue
		}
		bw := best.Bounds().Dx()
		switch {
		case w <= maxIconPx && (bw > maxIconPx || w > bw):
			best = img
		case bw > maxIconPx && w < bw:
			best = img
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, best); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
