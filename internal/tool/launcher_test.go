//go:build !windows

package tool

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The launcher reads the file in shell, for its `tool` verb, the frame
// and the install scripts. Two readers of one file drift unless a test
// holds them together: every text above must come out the same from both.
func TestTheLauncherAgrees(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash to run the launcher with")
	}
	launcher, err := filepath.Abs(filepath.Join("..", "..", "bin", Kit))
	if err != nil {
		t.Fatal(err)
	}
	// Looked at through os, so the test cache knows the script is an input
	// and runs this again when it changes.
	if _, err := os.Stat(launcher); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	run := func(file string) string {
		cmd := exec.Command(bash, launcher, "tool")
		cmd.Env = append(os.Environ(), Env+"="+file)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s tool: %v", launcher, err)
		}
		return string(out)
	}
	for name, text := range texts {
		file := filepath.Join(dir, File)
		if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, want := run(file), lines(Parse(text)); got != want {
			t.Errorf("%s: the launcher says\n%swant\n%s", name, got, want)
		}
	}
	if got, want := run(filepath.Join(dir, "none")), lines(Parse("")); got != want {
		t.Errorf("no file: the launcher says\n%swant\n%s", got, want)
	}
	big := "name mytool\n" + strings.Repeat("# padding\n", maxSize/10+1)
	file := filepath.Join(dir, File)
	if err := os.WriteFile(file, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, want := run(file), lines(Parse("")); got != want {
		t.Errorf("a file past the size limit: the launcher says\n%swant\n%s", got, want)
	}
}
