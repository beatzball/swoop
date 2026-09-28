// Package models lists what can answer in the Ask AI pane on this
// machine, for the two places that offer the choice: the AI model row
// in Settings and ctrl-k on New conversation. One list, so they agree.
package models

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/settings"
)

// Choice is one thing to pick: the value the ai setting takes, a title,
// and a note for the subtitle.
type Choice struct {
	Value string
	Title string
	Note  string
}

// Choices lists what is actually here: claude and codex when on PATH
// (and says so when not), each ollama model, each model LM Studio's
// server offers, openai with the model name typed after "openai:", and
// the line of your own. query is the bar's text, which may be that
// typed name.
func Choices(query string) []Choice {
	var cs []Choice
	if q := strings.TrimSpace(query); strings.HasPrefix(q, "openai:") && len(q) > len("openai:") {
		cs = append(cs, Choice{Value: q, Title: q, Note: "what you typed"})
	}
	for _, tool := range []struct{ name, note, missing string }{
		{"claude", "claude -p, streamed; searches the web when that is on", "not installed"},
		{"codex", "codex exec, answers whole; searches the web when that is on", "not installed"},
	} {
		if _, err := exec.LookPath(tool.name); err == nil {
			cs = append(cs, Choice{Value: tool.name, Title: tool.name, Note: tool.note})
		} else {
			cs = append(cs, Choice{Value: tool.name, Title: tool.name, Note: tool.missing})
		}
	}
	if out, err := exec.Command("ollama", "list").Output(); err == nil {
		for i, line := range strings.Split(string(out), "\n") {
			if i == 0 {
				continue
			}
			if f := strings.Fields(line); len(f) > 0 && !strings.Contains(f[0], "embed") {
				cs = append(cs, Choice{Value: "ollama:" + f[0], Title: "ollama " + f[0], Note: "local, streamed; no web"})
			}
		}
	}
	for _, m := range LMStudio() {
		cs = append(cs, Choice{Value: "lmstudio:" + m, Title: "lmstudio " + m, Note: "local, streamed; no web"})
	}
	cs = append(cs,
		Choice{Value: "openai:gpt-5", Title: "openai gpt-5", Note: "needs an API key; type openai:<model> for another"},
		Choice{Value: "", Title: "The line in ~/.config/swoop/ai", Note: "a command of your own; or the defaults when there is none"},
	)
	return cs
}

// Current is the ai setting in words: the value, or what the defaults
// would pick.
func Current() string {
	if v := settings.Get(settings.AI, ""); v != "" {
		return v
	}
	return "default: the line in ~/.config/swoop/ai, else claude, else ollama"
}

// LMStudio asks LM Studio's local server, when it is up, what models it
// has, embeddings left out. A server that does not answer within a
// moment has nothing to offer.
func LMStudio() []string {
	base := settings.Get(settings.AIURL, "http://localhost:1234/v1")
	if !strings.Contains(base, "localhost") && !strings.Contains(base, "127.0.0.1") {
		base = "http://localhost:1234/v1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/models", nil)
	if err != nil {
		return nil
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return nil
	}
	var ids []string
	for _, m := range out.Data {
		if !strings.Contains(m.ID, "embed") {
			ids = append(ids, m.ID)
		}
	}
	return ids
}
