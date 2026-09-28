//go:build darwin

package paste

import (
	"strings"
	"testing"
)

func TestAppleString(t *testing.T) {
	if got := appleString(`say "hi" \ bye`); got != `"say \"hi\" \\ bye"` {
		t.Errorf("quoted: %s", got)
	}
}

// No arrows unless asked; with them, the arrows come after the paste.
func TestKeystrokeScript(t *testing.T) {
	plain := strings.Join(keystrokeScript(0), "\n")
	if strings.Contains(plain, "key code 123") {
		t.Errorf("arrows with none asked:\n%s", plain)
	}
	back := strings.Join(keystrokeScript(3), "\n")
	paste, arrows := strings.Index(back, "keystroke \"v\""), strings.Index(back, "repeat 3 times\nkey code 123")
	if paste < 0 || arrows < paste {
		t.Errorf("want cmd+V, then three left arrows:\n%s", back)
	}
}
