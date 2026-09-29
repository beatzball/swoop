// Package models lists what can answer in the Ask AI pane on this
// machine, for the two places that offer the choice: the AI model row
// in Settings and ctrl-k on New conversation. One list, so they agree.
package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	for _, m := range Ollama() {
		cs = append(cs, Choice{Value: "ollama:" + m, Title: "ollama " + m, Note: "local, streamed; no web"})
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

// Ollama lists the models `ollama list` prints, newest pulled first, the
// way it prints them; nothing when ollama is not installed. Embedding
// models are left out: they cannot answer.
func Ollama() []string {
	out, err := exec.Command("ollama", "list").Output()
	if err != nil {
		return nil
	}
	return parseOllamaList(string(out))
}

// parseOllamaList takes the model names out of `ollama list`: the first
// column, after the header line.
func parseOllamaList(out string) []string {
	var names []string
	for i, line := range strings.Split(out, "\n") {
		if i == 0 {
			continue
		}
		if f := strings.Fields(line); len(f) > 0 && !strings.Contains(f[0], "embed") {
			names = append(names, f[0])
		}
	}
	return names
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
//
// A plain socket and a hand-written request rather than net/http: this
// is one GET to localhost, no TLS, and net/http brings the TLS stack
// with it, two megabytes in every tool that lists models. swoop-ai keeps
// net/http, since it talks to the API over TLS; this package is also in
// swoop-settings, which should not.
func LMStudio() []string {
	base := settings.Get(settings.AIURL, "http://localhost:1234/v1")
	if !strings.Contains(base, "localhost") && !strings.Contains(base, "127.0.0.1") {
		base = "http://localhost:1234/v1"
	}
	host, path := splitURL(base)
	body, ok := get(host, path+"/models", 800*time.Millisecond)
	if !ok {
		return nil
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &out) != nil {
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

// splitURL takes "http://host:port/v1" apart into "host:port" and "/v1".
func splitURL(base string) (host, path string) {
	rest := strings.TrimPrefix(strings.TrimPrefix(base, "http://"), "https://")
	rest = strings.TrimRight(rest, "/")
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i], rest[i:]
	}
	return rest, ""
}

// get is an HTTP/1.0 GET over a plain socket: the request written, the
// headers skipped, the body returned. Only for a server on this machine
// that answers in one piece, which LM Studio's model list is.
func get(host, path string, timeout time.Duration) ([]byte, bool) {
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.0\r\nHost: %s\r\nAccept: application/json\r\n\r\n", path, host); err != nil {
		return nil, false
	}
	resp, err := io.ReadAll(io.LimitReader(conn, 1<<20))
	if err != nil {
		return nil, false
	}
	head, body, found := bytes.Cut(resp, []byte("\r\n\r\n"))
	if !found || !bytes.HasPrefix(head, []byte("HTTP/1.")) || !bytes.Contains(head[:min(len(head), 16)], []byte(" 200")) {
		return nil, false
	}
	return body, true
}
