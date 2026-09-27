package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/beatzball/swoop/internal/settings"
)

// commandLine is the shell line that answers: the first line of
// ~/.config/swoop/ai, or a default from what is on PATH.
//
// The defaults, in order:
//
//	claude and jq   claude -p with its stream-json output through jq for
//	                the text as it arrives, so the answer streams
//	claude          claude -p, which answers in one piece
//
// Both claude forms allow WebSearch and WebFetch, which only read, when
// the web setting is on: without them a question about the weather ends
// in a refusal, since claude -p declines any tool it has not been given.
//
//	ollama          ollama run <the first model ollama lists>, which streams
//
// The command reads the conversation on stdin and writes the answer on
// stdout. That is the whole contract; nothing else is asked of it.
// jqFilter turns claude's stream-json into what the worker wants: the
// text on stdout as it arrives, and on stderr, for the line under the
// dots, each tool claude starts to use and the reason if it fails.
// jq's stderr builtin prints a string raw, with no newline of its own.
const jqFilter = `(select(.type == "stream_event") | .event.delta.text // empty),` +
	`(select(.type == "assistant") | .message.content[]? | select(.type == "tool_use") | "using \(.name)\n" | stderr | empty),` +
	`(select(.type == "result" and .is_error == true) | ((.result // .error // "the model returned an error") | tostring) + "\n" | stderr | empty)`

func commandLine() (string, error) {
	// The ai setting first: a preset by name, ollama with a model, or a
	// line of its own. Then the ai file. Then what is on PATH.
	switch choice := settings.Get(settings.AI, ""); {
	case choice == "claude":
		return claudeLine(), nil
	case strings.HasPrefix(choice, "ollama:"):
		return "ollama run " + strings.TrimPrefix(choice, "ollama:"), nil
	case choice != "":
		return choice, nil
	}
	if line := configured(); line != "" {
		return line, nil
	}
	if _, err := exec.LookPath("claude"); err == nil {
		return claudeLine(), nil
	}
	if _, err := exec.LookPath("ollama"); err == nil {
		if model := firstOllamaModel(); model != "" {
			return "ollama run " + model, nil
		}
	}
	return "", errors.New("no AI command found. Put one line in " + configPath() + ", for example: claude -p")
}

// claudeLine is the claude preset: streamed through jq when jq is there,
// and allowed to search and fetch the web when the web setting is on.
func claudeLine() string {
	tools := ""
	if settings.Get(settings.Web, settings.WebDefault) == "on" {
		tools = " --allowedTools WebSearch WebFetch"
	}
	if _, err := exec.LookPath("jq"); err == nil {
		// -j: no newline after each delta, they are pieces of text and
		// the model writes its own newlines. --unbuffered: each piece
		// out as it comes in.
		return `claude -p` + tools + ` --output-format stream-json --verbose --include-partial-messages | jq --unbuffered -rj '` + jqFilter + `'`
	}
	return "claude -p" + tools
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "swoop", "ai")
}

func configured() string {
	path := configPath()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

func firstOllamaModel() string {
	out, err := exec.Command("ollama", "list").Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return ""
	}
	fields := strings.Fields(lines[1])
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
