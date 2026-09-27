package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/beatzball/swoop/internal/settings"
)

// provider is what answers: a command with the conversation on its stdin
// and the answer on its stdout, or an HTTP API in the OpenAI chat shape.
// Two transports cover every preset; see the models design (#109).
type provider struct {
	name string // the preset, for messages: "claude", "lmstudio", "a line of your own"
	// The command kind.
	line string
	// The API kind.
	api       bool
	url       string // the base, ending in /v1
	model     string
	key       string
	responses bool // OpenAI's Responses API, which carries their web search tool
	// Whether this provider can search the web when the switch is on. When
	// it cannot, the status line says so.
	canSearch bool
}

const (
	openAIURL   = "https://api.openai.com/v1"
	lmStudioURL = "http://localhost:1234/v1"
)

// resolve reads the ai setting and turns it into a provider: a preset by
// name, a preset with a model after a colon, or a line of the user's own.
// With no setting, the line in ~/.config/swoop/ai, then claude if it is
// on PATH, then ollama's first model.
func resolve() (provider, error) {
	web := settings.Get(settings.Web, settings.WebDefault) == "on"
	choice := strings.TrimSpace(settings.Get(settings.AI, ""))
	kind, rest, _ := strings.Cut(choice, ":")
	switch {
	case choice == "":
		if line := configured(); line != "" {
			return provider{name: "the line in " + configPath(), line: line}, nil
		}
		if onPath("claude") {
			return provider{name: "claude", line: claudeLine(), canSearch: true}, nil
		}
		if onPath("ollama") {
			if model := firstOllamaModel(); model != "" {
				return provider{name: "ollama", line: ollamaLine(model)}, nil
			}
		}
		return provider{}, errors.New("no AI command found. Pick a model in Settings, or put one line in " + configPath() + ", for example: claude -p")
	case choice == "claude":
		if !onPath("claude") {
			return provider{}, errors.New("claude is not installed; see https://docs.anthropic.com/claude-code")
		}
		return provider{name: "claude", line: claudeLine(), canSearch: true}, nil
	case choice == "codex":
		if !onPath("codex") {
			return provider{}, errors.New("codex is not installed; npm install -g @openai/codex")
		}
		return provider{name: "codex", line: codexLine(web), canSearch: true}, nil
	case kind == "ollama" && rest != "":
		if !onPath("ollama") {
			return provider{}, errors.New("ollama is not installed; see https://ollama.com")
		}
		return provider{name: "ollama", line: ollamaLine(rest)}, nil
	case kind == "lmstudio" && rest != "":
		base := settings.Get(settings.AIURL, lmStudioURL)
		return provider{name: "lmstudio", api: true, url: base, model: rest, key: settings.Get(settings.AIKey, "lm-studio")}, nil
	case kind == "openai" && rest != "":
		base := settings.Get(settings.AIURL, openAIURL)
		key := settings.Get(settings.AIKey, os.Getenv("OPENAI_API_KEY"))
		if key == "" {
			return provider{}, errors.New("no API key: set it in Settings (API key), or OPENAI_API_KEY in the environment")
		}
		p := provider{name: "openai", api: true, url: base, model: rest, key: key}
		if isOpenAIHost(base) {
			p.canSearch = true
			p.responses = web
		}
		return p, nil
	}
	return provider{name: "a line of your own", line: choice}, nil
}

// webNote is what the status line says when the web switch is on but the
// provider has no way to search. Empty when the switch is off or it can.
func webNote(p provider) string {
	if settings.Get(settings.Web, settings.WebDefault) != "on" || p.canSearch {
		return ""
	}
	return "this model cannot search the web"
}

func isOpenAIHost(base string) bool {
	u, err := url.Parse(base)
	return err == nil && strings.EqualFold(u.Host, "api.openai.com")
}

// codexLine is the codex preset: a prompt on stdin, the answer whole on
// stdout, no shell commands (read-only sandbox), and with the switch on,
// codex's own live web search.
func codexLine(web bool) string {
	search := ""
	if web {
		search = " --search"
	}
	return fmt.Sprintf("codex%s exec -s read-only --skip-git-repo-check", search)
}

func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
