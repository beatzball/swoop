package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// commandLine is the shell line that answers: the first line of
// ~/.config/swoop/ai, or a default from what is on PATH.
//
// The defaults, in order:
//
//	claude and jq   claude -p with its stream-json output through jq for
//	                the text as it arrives, so the answer streams
//	claude          claude -p, which answers in one piece
//	ollama          ollama run <the first model ollama lists>, which streams
//
// The command reads the conversation on stdin and writes the answer on
// stdout. That is the whole contract; nothing else is asked of it.
func commandLine() (string, error) {
	if line := configured(); line != "" {
		return line, nil
	}
	if _, err := exec.LookPath("claude"); err == nil {
		if _, err := exec.LookPath("jq"); err == nil {
			// -j: no newline after each delta, they are pieces of text
			// and the model writes its own newlines. --unbuffered: each
			// piece out as it comes in.
			return `claude -p --output-format stream-json --verbose --include-partial-messages | jq --unbuffered -rj 'select(.type == "stream_event") | .event.delta.text // empty'`, nil
		}
		return "claude -p", nil
	}
	if _, err := exec.LookPath("ollama"); err == nil {
		if model := firstOllamaModel(); model != "" {
			return "ollama run " + model, nil
		}
	}
	return "", errors.New("no AI command found. Put one line in " + configPath() + ", for example: claude -p")
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
