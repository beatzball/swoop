package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lines is a Tool as the launcher's `tool` verb prints it.
func lines(t Tool) string {
	return "name " + t.Name + "\ntitle " + t.Title + "\nid " + t.ID + "\nhotkey " + t.Hotkey + "\n"
}

// texts are files a tool author could write, the good and the bad. Both
// readers, this package and the launcher script, are held to them.
var texts = map[string]string{
	"empty":         "",
	"all four":      "name mytool\ntitle My Tool\nid com.example.mytool\nhotkey alt+space\n",
	"name only":     "name mytool\n",
	"no last line":  "name mytool\nhotkey ctrl+space",
	"comments":      "# my tool\n\nname mytool\n  # indented\ntitle  Spaced  Out  \n",
	"tabs":          "name\tmytool\ntitle\tMy Tool\n",
	"windows lines": "name mytool\r\ntitle My Tool\r\nid com.example.mytool\r\n",
	"unknown key":   "name mytool\ncolour blue\n",
	"two words":     "name my tool\nid com example\nhotkey alt space\n",
	"a path":        "name ../elsewhere\nid /etc\n",
	"leading dot":   "name .hidden\nid -flag\n",
	"key alone":     "name\ntitle\n",
	"last one wins": "name one\nname two\n",
}

func TestParseFillsInWhatTheFileLeavesOut(t *testing.T) {
	got := Parse("")
	if got.Name != Kit || got.Title != Kit || got.ID != "dev."+Kit || got.Hotkey != hotkey {
		t.Fatalf("no file is the kit under its own name: %+v", got)
	}
	got = Parse(texts["name only"])
	want := Tool{Name: "mytool", Title: "mytool", ID: "dev.mytool", Hotkey: hotkey}
	if got != want {
		t.Fatalf("a name alone names the rest: got %+v, want %+v", got, want)
	}
	got = Parse(texts["all four"])
	want = Tool{Name: "mytool", Title: "My Tool", ID: "com.example.mytool", Hotkey: "alt+space"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseSkipsWhatCannotBeAPath(t *testing.T) {
	for _, name := range []string{"two words", "a path", "leading dot", "key alone"} {
		got := Parse(texts[name])
		if got.Name != Kit || got.ID != "dev."+Kit || got.Hotkey != hotkey {
			t.Errorf("%s: a value that is not one plain word must be skipped: %+v", name, got)
		}
	}
}

func TestParseReadsWindowsLinesTabsAndComments(t *testing.T) {
	got := Parse(texts["windows lines"])
	if got.Name != "mytool" || got.Title != "My Tool" || got.ID != "com.example.mytool" {
		t.Errorf("windows lines: %+v", got)
	}
	got = Parse(texts["tabs"])
	if got.Name != "mytool" || got.Title != "My Tool" {
		t.Errorf("tabs: %+v", got)
	}
	got = Parse(texts["comments"])
	if got.Name != "mytool" || got.Title != "Spaced  Out" {
		t.Errorf("comments: %+v", got)
	}
}

func TestReadTakesTheFileTheVariableNames(t *testing.T) {
	file := filepath.Join(t.TempDir(), File)
	if err := os.WriteFile(file, []byte(texts["all four"]), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(Env, file)
	if Path() != file {
		t.Fatalf("Path() = %q, want the file the variable names", Path())
	}
	if got := Name(); got != "mytool" {
		t.Fatalf("Name() = %q", got)
	}

	// A file that is not there is the kit under its own name, not an error.
	t.Setenv(Env, filepath.Join(t.TempDir(), "none"))
	if got := Read(); got != Parse("") {
		t.Fatalf("a missing file: %+v", got)
	}

	// Something large that happens to be called tool is not a tool file.
	big := "name mytool\n" + strings.Repeat("# padding\n", maxSize/10+1)
	if err := os.WriteFile(file, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(Env, file)
	if got := Read(); got != Parse("") {
		t.Fatalf("a file past the size limit: %+v", got)
	}
}

// With the variable unset, the file is beside the bin/ the program runs
// from. A test binary has none there, which is every test's default: the
// kit's own name.
func TestPathIsBesideTheProgramsFolder(t *testing.T) {
	t.Setenv(Env, "")
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if want := filepath.Join(filepath.Dir(exe), "..", File); Path() != want {
		t.Fatalf("Path() = %q, want %q", Path(), want)
	}
}
