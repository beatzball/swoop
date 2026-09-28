package paste

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Whatever the text, the frame reads back what was written: newlines,
// a line that looks like a count, emoji, nothing at all.
func TestRequestRoundTrip(t *testing.T) {
	for _, r := range []Request{
		{Text: "🚀"},
		{Text: "Best,\nSam\n", Left: 4},
		{Text: "12\n34", Left: 2},
		{Text: ""},
	} {
		path := filepath.Join(t.TempDir(), "paste")
		if err := WriteRequest(path, r); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeRequest(data)
		if err != nil || got != r {
			t.Errorf("wrote %+v, read %+v (%v)", r, got, err)
		}
		if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
			t.Errorf("the temporary file was left behind")
		}
	}
}

// The file format is the contract with the frame's Swift, so it is
// pinned here byte for byte.
func TestRequestFormat(t *testing.T) {
	if got := string(Request{Text: "a\nb", Left: 3}.Encode()); got != "3\na\nb" {
		t.Errorf("encoded %q", got)
	}
	if got := string(Request{Text: "x", Left: -1}.Encode()); got != "0\nx" {
		t.Errorf("a negative count: %q", got)
	}
	for _, bad := range []string{"", "3", "x\nhi", "-1\nhi"} {
		if _, err := DecodeRequest([]byte(bad)); err == nil {
			t.Errorf("%q decoded", bad)
		}
	}
}

// In the frame, Paste copies, leaves the request for the frame, and has
// nothing to say: the frame reports what it did.
func TestPasteInTheFrameWritesTheRequest(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no clipboard on this OS yet")
	}
	out := fakeClipboard(t)
	path := filepath.Join(t.TempDir(), "paste")
	t.Setenv("SWOOP_SHELL", "mac")
	t.Setenv(envRequest, path)
	note, err := PasteBack("Best,\n", 1)
	if err != nil || note != "" {
		t.Fatalf("note %q, err %v", note, err)
	}
	if got, _ := os.ReadFile(out); string(got) != "Best,\n" {
		t.Errorf("the clipboard got %q", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := DecodeRequest(data); err != nil || r != (Request{Text: "Best,\n", Left: 1}) {
		t.Errorf("request %+v (%v)", r, err)
	}
}

// A frame that does not paste sets no request path; there, as in a plain
// terminal, Enter copies and says so.
func TestPasteInAFrameWithoutARequestCopies(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no clipboard on this OS yet")
	}
	fakeClipboard(t)
	t.Setenv("SWOOP_SHELL", "mac")
	t.Setenv(envRequest, "")
	note, err := Paste("🚀")
	if err != nil || note == "" {
		t.Errorf("note %q, err %v", note, err)
	}
}
