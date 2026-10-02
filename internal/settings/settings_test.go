package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/tool"
)

func TestWeekStart(t *testing.T) {
	for value, want := range map[string]time.Weekday{
		"monday": time.Monday, "sunday": time.Sunday, " Sunday ": time.Sunday,
		// Anything else is the default, not a third kind of week.
		"": time.Monday, "saturday": time.Monday,
	} {
		if got := weekStart(value); got != want {
			t.Errorf("weekStart(%q) = %s, want %s", value, got, want)
		}
	}
}

func TestGetAndSetKeepOtherLines(t *testing.T) {
	text := "# mine\npreview = 60\n\nhotkey = alt+space\n"
	if got := get(text, "preview", "58"); got != "60" {
		t.Fatalf("get = %q", got)
	}
	if got := get(text, "missing", "x"); got != "x" {
		t.Fatalf("fallback = %q", got)
	}
	out := set(text, "preview", "65")
	want := "# mine\npreview = 65\n\nhotkey = alt+space\n"
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
	data, _ := os.ReadFile(filepath.Join(dir, tool.Name(), "config"))
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

func TestParseOff(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"   ", ""},
		{"reminders", "reminders"},
		{" reminders ,tasks,  emoji ", "reminders|tasks|emoji"},
		{"reminders,,tasks,", "reminders|tasks"},
		{"tasks, tasks", "tasks"},
		// Settings is the way back, so it is never off.
		{"settings, window", "window"},
		// A name nothing answers to is kept: it may be installed later.
		{"nosuch", "nosuch"},
	}
	for _, c := range cases {
		got := strings.Join(ParseOff(c.in), "|")
		if got != c.want {
			t.Errorf("ParseOff(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := FormatOff([]string{"tasks", " reminders", "settings", "tasks"}); got != "tasks, reminders" {
		t.Errorf("FormatOff = %q", got)
	}
}

func TestOffListFromFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := OffList(); len(got) != 0 {
		t.Fatalf("no file: %v", got)
	}
	if err := Set(Off, "reminders , tasks"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(OffList(), "|"); got != "reminders|tasks" {
		t.Fatalf("OffList = %q", got)
	}
}

func TestEditorCommand(t *testing.T) {
	for _, c := range []struct {
		setting, env string
		want         []string
	}{
		{"nvim", "vim", []string{"nvim"}},
		{"emacs -nw", "", []string{"emacs", "-nw"}},
		{"", "hx", []string{"hx"}},
		{"  ", " code  --wait ", []string{"code", "--wait"}},
		{"", "", []string{"nano"}},
	} {
		got := editorCommand(c.setting, c.env)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("editorCommand(%q, %q) = %q, want %q", c.setting, c.env, got, c.want)
		}
	}
}
