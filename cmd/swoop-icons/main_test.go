package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/bundle"
	"github.com/beatzball/swoop/internal/protocol"
)

// TestRunSkipsAndWarms feeds run a fake lookup, so it needs no app
// bundles: a cached icon becomes picture cells, a not-cached one keeps its
// glyph and is named for the warmer, and an app with no icon at all keeps
// its glyph and is not, since warming would never help it.
func TestRunSkipsAndWarms(t *testing.T) {
	rows := []protocol.Item{
		{ID: "/A/Cached.app", Kind: "app", Icon: "x", Title: "Cached"},
		{ID: "/A/Cold.app", Kind: "app", Icon: "x", Title: "Cold"},
		{ID: "/A/Bare.app", Kind: "app", Icon: "x", Title: "Bare"},
		{ID: "calc", Kind: "text", Icon: "=", Title: "4"},
	}
	var in strings.Builder
	for _, it := range rows {
		in.WriteString(it.Line() + "\n")
	}
	lookup := func(app string) ([]byte, error) {
		switch app {
		case "/A/Cached.app":
			return []byte("png"), nil
		case "/A/Cold.app":
			return nil, bundle.ErrNotCached
		}
		return nil, bundle.ErrNoIcon
	}

	var out, pics bytes.Buffer
	missing, err := run(strings.NewReader(in.String()), &out, &pics, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"/A/Cold.app"}; !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %q, want %q", missing, want)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != len(rows) {
		t.Fatalf("got %d lines, want %d", len(lines), len(rows))
	}
	icons := make([]string, len(lines))
	for i, l := range lines {
		it, err := protocol.Parse(l)
		if err != nil {
			t.Fatal(err)
		}
		icons[i] = it.Icon
	}
	if icons[0] == "x " {
		t.Errorf("cached icon kept its glyph")
	}
	for i, want := range []string{"x ", "x ", "= "} {
		if got := icons[i+1]; got != want {
			t.Errorf("row %d icon = %q, want %q", i+1, got, want)
		}
	}
	if pics.Len() == 0 {
		t.Errorf("no picture sent for the cached icon")
	}
}

// TestRunNothingToWarmWithoutPictures checks that when the pictures have
// nowhere to go, no icon is looked up and nothing is warmed.
func TestRunNothingToWarmWithoutPictures(t *testing.T) {
	line := protocol.Item{ID: "/A/Cold.app", Kind: "app", Icon: "x", Title: "Cold"}.Line() + "\n"
	lookup := func(string) ([]byte, error) { return nil, errors.New("looked up") }
	var out bytes.Buffer
	missing, err := run(strings.NewReader(line), &out, nil, lookup)
	if err != nil || len(missing) != 0 {
		t.Errorf("missing = %q, err = %v, want none", missing, err)
	}
}
