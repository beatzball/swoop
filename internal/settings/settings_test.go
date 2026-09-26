package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAndSetKeepOtherLines(t *testing.T) {
	text := "# swoop\npreview = 60\n\nhotkey = alt+space\n"
	if got := get(text, "preview", "58"); got != "60" {
		t.Fatalf("get = %q", got)
	}
	if got := get(text, "missing", "x"); got != "x" {
		t.Fatalf("fallback = %q", got)
	}
	out := set(text, "preview", "65")
	want := "# swoop\npreview = 65\n\nhotkey = alt+space\n"
	if out != want {
		t.Fatalf("set replaced wrong:\n%q\nwant\n%q", out, want)
	}
	out = set(text, "theme", "dark")
	if out != text+"theme = dark\n" {
		t.Fatalf("set should append a new key:\n%q", out)
	}
	if out := set("", "preview", "40"); out != "preview = 40\n" {
		t.Fatalf("set on an empty file: %q", out)
	}
}

func TestPreviewPercentFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if got := PreviewPercent(); got != PreviewDefault {
		t.Fatalf("no file: %d", got)
	}
	if err := Set(Preview, "70"); err != nil {
		t.Fatal(err)
	}
	if got := PreviewPercent(); got != 70 {
		t.Fatalf("after Set: %d", got)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "swoop", "config"))
	if string(data) != "preview = 70\n" {
		t.Fatalf("file: %q", data)
	}
	_ = Set(Preview, "95")
	if got := PreviewPercent(); got != PreviewMax {
		t.Fatalf("clamped high: %d", got)
	}
	_ = Set(Preview, "junk")
	if got := PreviewPercent(); got != PreviewDefault {
		t.Fatalf("junk falls back: %d", got)
	}
}
