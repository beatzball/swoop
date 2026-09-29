package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/beatzball/swoop/internal/models"
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
	// note is said in the status line before the model says anything: for
	// a default that picked a model, which one and why.
	note string
}

const (
	openAIURL   = "https://api.openai.com/v1"
	lmStudioURL = "http://localhost:1234/v1"
)

// resolve reads the ai setting and turns it into a provider: a preset by
// name, a preset with a model after a colon, or a line of the user's own.
// With no setting, the line in ~/.config/swoop/ai, then claude if it is
// on PATH, then ollama with the model pickOllama chooses.
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
			if model, why := pickOllama(models.Ollama(), settings.Get(settings.Ollama, "")); model != "" {
				return provider{name: "ollama", line: ollamaLine(model), note: why}, nil
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
	case choice == "ollama":
		// ollama with no model named: the same pick as the defaults.
		if !onPath("ollama") {
			return provider{}, errors.New("ollama is not installed; see https://ollama.com")
		}
		model, why := pickOllama(models.Ollama(), settings.Get(settings.Ollama, ""))
		if model == "" {
			return provider{}, errors.New("ollama has no models; ollama pull one, for example: ollama pull llama3.2")
		}
		return provider{name: "ollama", line: ollamaLine(model), note: why}, nil
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

// startNote is the status line before the model says anything: the
// provider's own note and the web note, whichever there are. One line,
// since the pane shows only the last thing said.
func startNote(p provider) string {
	var notes []string
	for _, n := range []string{p.note, webNote(p)} {
		if n != "" {
			notes = append(notes, n)
		}
	}
	return strings.Join(notes, " · ")
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
