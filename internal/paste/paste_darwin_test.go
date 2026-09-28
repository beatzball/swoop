//go:build darwin

package paste

import "testing"

func TestAppleString(t *testing.T) {
	if got := appleString(`say "hi" \ bye`); got != `"say \"hi\" \\ bye"` {
		t.Errorf("quoted: %s", got)
	}
}
