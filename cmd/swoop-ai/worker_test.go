package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/settings"
)

func TestOutcome(t *testing.T) {
	if text, err := outcome("Blue.", "", nil); text != "Blue." || err != nil {
		t.Fatalf("an answer is an answer: %q %v", text, err)
	}
	if text, err := outcome("Blue.", "warning: slow", errors.New("exit 1")); text != "Blue." || err != nil {
		t.Fatalf("an answer with a bad exit is still an answer: %q %v", text, err)
	}
	if _, err := outcome("", "using WebSearch\nrate limit reached", nil); err == nil || err.Error() != "rate limit reached" {
		t.Fatalf("no text with exit 0 is a failure, told by stderr's last line: %v", err)
	}
	if _, err := outcome("  \n", "", errors.New("exit 1")); err == nil || err.Error() != "exit 1" {
		t.Fatalf("no text, no stderr: the exit error: %v", err)
	}
	if _, err := outcome("", "", nil); err == nil || err.Error() != "the command printed nothing" {
		t.Fatalf("nothing at all: %v", err)
	}
}

func TestStripANSI(t *testing.T) {
	in := "I need to re\x1b[3D\x1b[Kreal-time data\x1b[?25l \u283c \x1b[K\x1b[?25h"
	if got := stripANSI(in); got != "I need to rereal-time data  " {
		t.Fatalf("got %q", got)
	}
	if got := stripANSI("plain words"); got != "plain words" {
		t.Fatalf("plain text untouched: %q", got)
	}
	if got := stripANSI("87°F. \ue200cite\ue202turn2search8\ue201 done"); got != "87°F.  done" {
		t.Fatalf("codex citation marks go: %q", got)
	}
}

func TestOllamaLineKeepsTheAnswerClean(t *testing.T) {
	got := ollamaLine("ornith-1.5:9b")
	for _, flag := range []string{"--nowordwrap", "--hidethinking", "--think=false"} {
		if !strings.Contains(got, flag) {
			t.Errorf("missing %s in %q", flag, got)
		}
	}
	if !strings.HasSuffix(got, " ornith-1.5:9b") {
		t.Errorf("the model last: %q", got)
	}
}

func TestPickOllama(t *testing.T) {
	listed := []string{"qwen3:8b", "llama3.2:latest"}
	for _, c := range []struct {
		named, model, why string
	}{
		{"", "qwen3:8b", "the newest pulled"},
		{"llama3.2", "llama3.2:latest", "named in settings"},
		{"llama3.2:latest", "llama3.2:latest", "named in settings"},
		{"mistral", "qwen3:8b", "mistral is not pulled"},
	} {
		model, why := pickOllama(listed, c.named)
		if model != c.model || !strings.Contains(why, c.why) || !strings.Contains(why, model) {
			t.Errorf("named %q: got %q, %q; want %q, a note saying %q and the model", c.named, model, why, c.model, c.why)
		}
	}
	if model, why := pickOllama(nil, "llama3.2"); model != "" || why != "" {
		t.Errorf("no models, no pick: %q %q", model, why)
	}
}

func TestStartNoteSaysBoth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p := provider{note: "ollama qwen3:8b, the newest pulled"}
	if got := startNote(p); got != p.note {
		t.Errorf("web off: only the pick: %q", got)
	}
	if err := settings.Set(settings.Web, "on"); err != nil {
		t.Fatal(err)
	}
	if got := startNote(p); !strings.Contains(got, "qwen3:8b") || !strings.Contains(got, "cannot search the web") {
		t.Errorf("web on, a model that cannot: both notes in one line: %q", got)
	}
	if got := startNote(provider{canSearch: true}); got != "" {
		t.Errorf("nothing to say: %q", got)
	}
}

func TestCopyRowsSayWhatIsMissing(t *testing.T) {
	rows := conversationActions("")
	if rows[0].Subtitle != "Enter" {
		t.Errorf("copy works: the key: %q", rows[0].Subtitle)
	}
	rows = conversationActions("no clipboard tool: install wl-clipboard or xclip")
	for _, r := range rows[:2] {
		if !strings.Contains(r.Subtitle, "xclip") {
			t.Errorf("%s: the reason copy cannot work belongs in the row: %q", r.Title, r.Subtitle)
		}
	}
}
