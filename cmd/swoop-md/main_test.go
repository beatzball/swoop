package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/markdown"
)

func TestWidthOr(t *testing.T) {
	t.Setenv("FZF_PREVIEW_COLUMNS", "")
	t.Setenv("COLUMNS", "")
	if got := widthOr(0); got != 80 {
		t.Fatalf("nothing set: %d", got)
	}
	t.Setenv("COLUMNS", "100")
	if got := widthOr(0); got != 100 {
		t.Fatalf("COLUMNS: %d", got)
	}
	t.Setenv("FZF_PREVIEW_COLUMNS", "45")
	if got := widthOr(0); got != 45 {
		t.Fatalf("fzf's width wins over COLUMNS: %d", got)
	}
	if got := widthOr(30); got != 30 {
		t.Fatalf("the flag wins over everything: %d", got)
	}
}

// The command is the library behind a pipe: the same markdown through
// both gives the same bytes.
func TestCommandIsTheLibrary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("builds a binary and runs it through a pipe; skipped on Windows")
	}
	bin := filepath.Join(t.TempDir(), "swoop-md")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	md := "# Title\n\nSome **bold** text that is long enough to wrap at thirty columns.\n\n- one\n- two\n"
	cmd := exec.Command(bin, "-w", "30")
	cmd.Stdin = strings.NewReader(md)
	cmd.Env = append(os.Environ(), "FZF_PREVIEW_COLUMNS=", "COLUMNS=")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("swoop-md: %v", err)
	}
	if string(out) != markdown.Render(md, 30) {
		t.Fatalf("command differs from the library:\n%q\nvs\n%q", out, markdown.Render(md, 30))
	}
}
