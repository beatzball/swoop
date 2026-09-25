package chat

import (
	"strings"
	"testing"
	"time"
)

func TestNewAskSaveLoadListDelete(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	c, err := s.New("why is the sky blue, in one line, please, and keep it short enough for a title")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(c.Title, "…") || len(c.Title) > TitleMax+len("…") {
		t.Fatalf("title should be cut: %q", c.Title)
	}
	c.Ask("why is the sky blue")
	if !c.Pending || len(c.Turns) != 2 || c.Turns[1].Role != "assistant" || c.Turns[1].Text != "" {
		t.Fatalf("after Ask: %+v", c)
	}
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Turns[0].Text != "why is the sky blue" || !got.Pending {
		t.Fatalf("loaded: %+v", got)
	}

	time.Sleep(2 * time.Millisecond)
	d, _ := s.New("second")
	d.Ask("second")
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	all, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != d.ID {
		t.Fatalf("newest first: %v", all)
	}
	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	all, _ = s.List()
	if len(all) != 1 {
		t.Fatalf("after delete: %v", all)
	}
	if err := s.Delete(c.ID); err != nil {
		t.Fatalf("deleting twice is fine: %v", err)
	}
}

func TestIDsCannotEscapeTheStore(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	for _, id := range []string{"", "../x", "a/b", ".hidden", `a\b`} {
		if _, err := s.Load(id); err != ErrBadID {
			t.Errorf("Load(%q) = %v, want ErrBadID", id, err)
		}
		if err := s.Delete(id); err != ErrBadID {
			t.Errorf("Delete(%q) = %v, want ErrBadID", id, err)
		}
	}
}

func TestPromptIsBareForOneTurnAndFramedAfter(t *testing.T) {
	c := &Conversation{}
	c.Ask("hello")
	if got := c.Prompt(); got != "hello\n" {
		t.Fatalf("first prompt goes alone: %q", got)
	}
	c.Turns[1].Text = "hi there"
	c.Pending = false
	c.Ask("and again")
	got := c.Prompt()
	for _, part := range []string{"User: hello", "Assistant: hi there", "User: and again"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
	if c.Answer() != "" {
		t.Fatalf("the pending answer is empty, got %q", c.Answer())
	}
	c.Turns[3].Text = "sure"
	if c.Answer() != "sure" {
		t.Fatalf("Answer = %q", c.Answer())
	}
}
