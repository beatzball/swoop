package ext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// beside writes a file next to the extension name's executable in dir.
func beside(t *testing.T, dir, name, file, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name, file), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidKey(t *testing.T) {
	for _, k := range []string{"tab", "shift-tab", "f1", "f12", "alt-a", "alt-z", "alt-0", "alt-,", "alt-.", "alt-/"} {
		if !ValidKey(k) {
			t.Errorf("%q may be claimed", k)
		}
	}
	// The launcher's keys, the bar's, and anything a shell or a binding
	// would read as its own.
	for _, k := range []string{"", "enter", "esc", "up", "ctrl-k", "ctrl-a", "alt-left", "alt-A", "alt-;", "alt-'", "alt-", "alt-ab", "f0", "f13", "f01", "tab;rm", "TAB"} {
		if ValidKey(k) {
			t.Errorf("%q may not be claimed", k)
		}
	}
}

func TestClaimsReadTheKeyFile(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "ask", "")
	fake(t, dir, "plain", "")
	beside(t, dir, "ask", KeyFile, "# the way in\n\ntab ask Ask Something\nalt-k other\nnonsense\nctrl-k ask Taken\n")
	exts := Discover([]string{dir})
	got := exts[0].Claims()
	want := []Claim{{"tab", "ask", "Ask Something"}, {"alt-k", "other", "ask"}, {"ctrl-k", "ask", "Taken"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("claim %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if c := exts[1].Claims(); c != nil {
		t.Fatalf("no file is no claim: %+v", c)
	}
	// A key nobody may claim is bound to nothing.
	if keys := Keys(exts); strings.Join(keys, " ") != "tab alt-k" {
		t.Fatalf("Keys = %v", keys)
	}
	if _, _, ok := ByKey(exts, "ctrl-k"); ok {
		t.Fatal("ctrl-k is the launcher's")
	}
}

// Two claim one key: the first by name has it, the report says so, and
// with that one gone the next has it.
func TestByKeyTheFirstByNameWins(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "bravo", "")
	fake(t, dir, "alpha", "")
	beside(t, dir, "alpha", KeyFile, "tab ask Alpha\n")
	beside(t, dir, "bravo", KeyFile, "tab ask Bravo\nf5 other Bravo Other\nctrl-c x Nope\n")
	exts := Discover([]string{dir})
	e, c, ok := ByKey(exts, "tab")
	if !ok || e.Name != "alpha" || c.Title != "Alpha" {
		t.Fatalf("ByKey(tab) = %+v %+v %v", e, c, ok)
	}
	if e, _, ok := ByKey(exts[1:], "tab"); !ok || e.Name != "bravo" {
		t.Fatalf("without alpha: %+v %v", e, ok)
	}
	if keys := Keys(exts); strings.Join(keys, " ") != "tab f5" {
		t.Fatalf("Keys = %v", keys)
	}
	want := []string{
		"tab opens Alpha, from alpha",
		"tab: bravo claims it too, and alpha has it, the first by name",
		"f5 opens Bravo Other, from bravo",
		"ctrl-c: not a key an extension may claim, asked for by bravo",
	}
	if got := KeyReport(exts); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(got, "\n"))
	}
}

func TestPaneReadsTheViewsFile(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fake(t, dir, "ask", "")
	beside(t, dir, "ask", ViewsFile, "# panes\nask bar=prompt preview=wrap,follow\nwide preview=70%\nhuge preview=95%,wrap later=thing\nodd bar=filter preview=left\n")
	e := Discover([]string{dir})[0]
	cases := map[string]Pane{
		"ask":  {Prompt: true, Wrap: true, Follow: true},
		"wide": {Percent: 70},
		// Kept in the range the user's own width has, and a setting from
		// a newer launcher is skipped.
		"huge":    {Percent: 80, Wrap: true},
		"odd":     {},
		"missing": {},
	}
	for view, want := range cases {
		if got := e.Pane(view); got != want {
			t.Errorf("Pane(%q) = %+v, want %+v", view, got, want)
		}
	}
	fake(t, dir, "plain", "")
	if got := Discover([]string{dir})[1].Pane("ask"); got != (Pane{}) {
		t.Fatalf("no file is a plain list: %+v", got)
	}
}

func TestSendPassesTheRowAndTheText(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	fake(t, dir, "ask", `[ "$1" = send ] || exit 2
[ "$3" != refuse ] || exit 1
printf '%s|%s' "$2" "$3" > '`+out+`'
`)
	e := Discover([]string{dir})[0]
	if err := e.Send("new", "why is the sky blue"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(out); string(data) != "new|why is the sky blue" {
		t.Fatalf("send got %q", data)
	}
	if err := e.Send("new", "refuse"); err == nil {
		t.Fatal("a non-zero exit is the text not taken")
	}
}

// TestBundledClaims holds the keys and panes the repository's extensions
// ship with, so a missing or renamed file is caught here and not by a
// user whose Tab does nothing.
func TestBundledClaims(t *testing.T) {
	skipOnWindows(t)
	exts := DiscoverAll([]string{filepath.Join("..", "..", "extensions")})
	want := []string{
		"tab opens Ask AI, from ai",
		"alt-, opens Settings, from settings",
	}
	if got := KeyReport(exts); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(got, "\n"))
	}
	e, c, ok := ByKey(exts, "tab")
	if !ok || c.View != "ask" {
		t.Fatalf("tab: %+v %v", c, ok)
	}
	if got := e.Pane(c.View); got != (Pane{Prompt: true, Wrap: true, Follow: true}) {
		t.Fatalf("the ask view's pane: %+v", got)
	}
	e, c, _ = ByKey(exts, "alt-,")
	if got := e.Pane(c.View); c.View != "settings" || got != (Pane{}) {
		t.Fatalf("the settings view: %+v %+v", c, got)
	}
}
