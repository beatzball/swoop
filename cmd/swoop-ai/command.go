package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/beatzball/swoop/internal/settings"
)

// jqFilter turns claude's stream-json into what the worker wants: the
// text on stdout as it arrives, and on stderr, for the line under the
// dots, each tool claude starts to use and the reason if it fails.
// jq's stderr builtin prints a string raw, with no newline of its own.
const jqFilter = `(select(.type == "stream_event") | .event.delta.text // empty),` +
	`(select(.type == "assistant") | .message.content[]? | select(.type == "tool_use") | "using \(.name)\n" | stderr | empty),` +
	`(select(.type == "result" and .is_error == true) | ((.result // .error // "the model returned an error") | tostring) + "\n" | stderr | empty)`

// ollamaLine is the ollama preset for one model. --nowordwrap: through a
// pipe ollama still wraps words with cursor moves, which would land in
// the transcript. --hidethinking and --think=false: a thinking model's
// reasoning is not the answer, and a model without thinking takes the
// flags without complaint.
func ollamaLine(model string) string {
	return "ollama run --nowordwrap --hidethinking --think=false " + model
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
