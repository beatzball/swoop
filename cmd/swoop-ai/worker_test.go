package main

import (
	"errors"
	"strings"
	"testing"
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
