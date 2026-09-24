package clip

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreDefaultsWhenMissing(t *testing.T) {
	m, err := LoadIgnore(filepath.Join(t.TempDir(), "none"))
	if err != nil || !m["com.1password.1password"] {
		t.Fatalf("missing file should give the defaults: %v %v", m, err)
	}
}

func TestLoadIgnoreReadsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.ignore")
	if err := os.WriteFile(path, []byte("# my list\n\nCom.Example.Vault\n  com.other.app  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := LoadIgnore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !m["com.example.vault"] || !m["com.other.app"] || len(m) != 2 {
		t.Fatalf("got %v", m)
	}
	if m["com.1password.1password"] {
		t.Fatal("a user's list replaces the defaults; it does not add to them")
	}
}

func TestLoadIgnoreEmptyFileIgnoresNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.ignore")
	if err := os.WriteFile(path, []byte("\n# nothing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := LoadIgnore(path)
	if err != nil || len(m) != 0 {
		t.Fatalf("got %v %v", m, err)
	}
}
