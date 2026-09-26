package main

import (
	"errors"
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
