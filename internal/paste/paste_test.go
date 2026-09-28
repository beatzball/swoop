package paste

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeClipboard puts a pbcopy, wl-copy and xclip first on PATH that write
// what they are given to a file, and returns that file.
func fakeClipboard(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "copied")
	for _, name := range []string{"pbcopy", "wl-copy", "xclip"} {
		body := "#!/bin/sh\ncat > '" + out + "'\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return out
}

// Outside the frame, Paste copies and says to paste by hand: a keystroke
// would land in the terminal the launcher ran in.
func TestPasteOutsideTheFrameCopies(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no clipboard on this OS yet")
	}
	out := fakeClipboard(t)
	t.Setenv("SWOOP_SHELL", "")
	note, err := Paste("🚀")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "Copied 🚀") || !strings.Contains(note, pasteKey) {
		t.Errorf("note: %q", note)
	}
	got, _ := os.ReadFile(out)
	if string(got) != "🚀" {
		t.Errorf("the clipboard got %q", got)
	}
}
