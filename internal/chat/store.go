// Package chat keeps the conversations of the Ask AI pane: one JSON file
// each, under the user's state directory. Two processes touch a
// conversation at once, the worker writing the answer as it streams and
// the preview reading it on every refresh, so every write is a temp file
// and a rename, and a reader never sees half a file.
package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Turn is one side of one exchange.
type Turn struct {
	Role string `json:"role"` // "user" or "assistant"
	Text string `json:"text"`
}

// Conversation is the file. While Pending, the last turn is the
// assistant's and its Text grows as the model writes; Tick counts the
// worker's refreshes so the preview can animate while Text is empty.
type Conversation struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Turns   []Turn    `json:"turns"`
	Pending bool      `json:"pending,omitempty"`
	Tick    int       `json:"tick,omitempty"`
	Error   string    `json:"error,omitempty"`
	// Asked is when the pending prompt was sent, for the clock under the
	// dots; Status the last line the command wrote to stderr, shown
	// there too; Worker the pid of the process answering, so a pane can
	// tell a worker that died from one still thinking.
	Asked  time.Time `json:"asked,omitempty"`
	Status string    `json:"status,omitempty"`
	Worker int       `json:"worker,omitempty"`
}

// Store is the directory.
type Store struct{ Dir string }

// DefaultDir is $XDG_STATE_HOME/swoop/ai, or ~/.local/state/swoop/ai.
func DefaultDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "swoop", "ai")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "swoop", "ai")
}

// TitleMax is how much of the first prompt becomes the title.
const TitleMax = 60

// New starts a conversation from its first prompt and saves it.
func (s Store) New(prompt string) (*Conversation, error) {
	now := time.Now()
	c := &Conversation{
		ID:      now.Format("20060102-150405") + "-" + suffix(),
		Title:   title(prompt),
		Created: now,
	}
	return c, nil
}

func title(prompt string) string {
	t := strings.Join(strings.Fields(prompt), " ")
	if len(t) > TitleMax {
		t = strings.TrimSpace(t[:TitleMax]) + "…"
	}
	return t
}

func suffix() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b)
}

func (s Store) path(id string) string {
	return filepath.Join(s.Dir, id+".json")
}

// ErrBadID rejects an id that could name a file outside the store.
var ErrBadID = errors.New("chat: bad conversation id")

func validID(id string) bool {
	return id != "" && !strings.ContainsAny(id, "/\\") && !strings.HasPrefix(id, ".")
}

// Load reads one conversation.
func (s Store) Load(id string) (*Conversation, error) {
	if !validID(id) {
		return nil, ErrBadID
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	c := &Conversation{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("chat: %s: %w", id, err)
	}
	return c, nil
}

// Save writes one conversation, whole or not at all.
func (s Store) Save(c *Conversation) error {
	if !validID(c.ID) {
		return ErrBadID
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	c.Updated = time.Now()
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, "."+c.ID+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path(c.ID))
}

// Delete removes one conversation. A missing one is already deleted.
func (s Store) Delete(id string) error {
	if !validID(id) {
		return ErrBadID
	}
	err := os.Remove(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// List returns every conversation, most recently updated first. A file
// that does not parse is skipped, not fatal.
func (s Store) List() ([]*Conversation, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var all []*Conversation
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".") {
			continue
		}
		c, err := s.Load(strings.TrimSuffix(name, ".json"))
		if err != nil {
			continue
		}
		all = append(all, c)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Updated.After(all[j].Updated) })
	return all, nil
}

// Ask appends the user's prompt and an empty answer for the model to fill,
// and marks the conversation pending.
func (c *Conversation) Ask(prompt string) {
	c.Turns = append(c.Turns, Turn{Role: "user", Text: prompt}, Turn{Role: "assistant"})
	c.Pending = true
	c.Tick = 0
	c.Error = ""
	c.Asked = time.Now()
	c.Status = ""
	c.Worker = 0
}

// Answer is the text of the last assistant turn, or "".
func (c *Conversation) Answer() string {
	for i := len(c.Turns) - 1; i >= 0; i-- {
		if c.Turns[i].Role == "assistant" {
			return c.Turns[i].Text
		}
	}
	return ""
}

// Prompt is what the model reads on stdin: the whole conversation as
// plain text, User and Assistant blocks, the new prompt last. A first
// prompt goes alone, with no framing to confuse a model that expects a
// question. The model command knows nothing of swoop, so this is the one
// format every command gets.
func (c *Conversation) Prompt() string {
	var turns []Turn
	for _, t := range c.Turns {
		if t.Text != "" {
			turns = append(turns, t)
		}
	}
	if len(turns) == 1 {
		return turns[0].Text + "\n"
	}
	var b strings.Builder
	b.WriteString("This is a conversation so far. Answer the last User message.\n\n")
	for _, t := range turns {
		switch t.Role {
		case "user":
			b.WriteString("User: ")
		default:
			b.WriteString("Assistant: ")
		}
		b.WriteString(strings.TrimSpace(t.Text))
		b.WriteString("\n\n")
	}
	return b.String()
}

// Transcript is the conversation as plain text for copying.
func (c *Conversation) Transcript() string {
	var b strings.Builder
	for _, t := range c.Turns {
		if t.Role == "user" {
			b.WriteString("You: ")
		} else {
			b.WriteString("AI: ")
		}
		b.WriteString(strings.TrimSpace(t.Text))
		b.WriteString("\n\n")
	}
	return b.String()
}
