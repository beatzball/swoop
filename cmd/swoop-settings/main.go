// swoop-settings is the Settings pane: the extension behind the Settings
// row and cmd+, in the frame. It speaks the extension contract, so the
// launcher routes to it like any other.
//
//	swoop-settings list                 the Settings row
//	swoop-settings view settings [q]    one row per setting, the value in the subtitle
//	swoop-settings view <key> [q]       the choices for one setting
//	swoop-settings preview <id>         what the setting does and where it lives
//	swoop-settings run <id>             <key>=<value> sets it; folder opens the config folder
//
// Every value is one line in ~/.config/swoop/config, the file
// internal/settings reads and writes. The frame watches that file for the
// hotkey, so a change here takes effect with no restart.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/models"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
)

// A setting the pane shows: its key in the config file, its title, how
// to describe the current value, and its choices.
type setting struct {
	key     string
	title   string
	icon    string
	explain string
	value   func() string // the subtitle: the current value, in words
	choices func(query string) []choice
}

type choice struct {
	value string // what goes in the file
	title string
	note  string
}

var all = []setting{
	{
		key: settings.Hotkey, title: "Hotkey", icon: "󰌌",
		explain: "The key that opens swoop, in the frame. The frame watches the settings\nfile and takes a new key at once. A key another app holds cannot be\ntaken; the frame's log says so.",
		value:   func() string { return settings.Get(settings.Hotkey, settings.HotkeyDefault) },
		choices: hotkeyChoices,
	},
	{
		key: settings.Preview, title: "Preview width", icon: "󰤼",
		explain: "How much of the window the preview takes. The divider keys, cmd+[ and\ncmd+] in the frame, move it 5% a step too.",
		value:   func() string { return strconv.Itoa(settings.PreviewPercent()) + "%" },
		choices: func(string) []choice {
			var cs []choice
			for p := settings.PreviewMin; p <= settings.PreviewMax; p += 5 {
				cs = append(cs, choice{value: strconv.Itoa(p), title: strconv.Itoa(p) + "%", note: ""})
			}
			return cs
		},
	},
	{
		key: settings.AI, title: "AI model", icon: "󰭹",
		explain: "What answers in the Ask AI pane. A preset by name, ollama with a model,\nor a line of your own in ~/.config/swoop/ai: a command that reads the\nconversation on stdin and prints the answer.",
		value:   aiValue,
		choices: aiChoices,
	},
	{
		key: settings.Web, title: "Web search for AI", icon: "󰖟",
		explain: "Whether the model may search and fetch the web. claude adds its\nWebSearch and WebFetch tools when this is on. Off, a question about your\nown text stays on your Mac.",
		value: func() string {
			if settings.Get(settings.Web, settings.WebDefault) == "on" {
				return "on · claude, codex and openai search; local models cannot"
			}
			return "off"
		},
		choices: func(string) []choice {
			return []choice{{value: "on", title: "On", note: "claude, codex and openai may search; local models cannot"}, {value: "off", title: "Off", note: "nothing leaves for the web"}}
		},
	},
	{
		key: settings.AIURL, title: "API URL", icon: "󰖟",
		explain: "Where openai:<model> and lmstudio:<model> send their requests: the base\nURL ending in /v1. OpenAI's own, LM Studio's, or any server with that\nAPI: OpenRouter, Groq, vLLM. Type one to use it.",
		value: func() string {
			if v := settings.Get(settings.AIURL, ""); v != "" {
				return v
			}
			return "default: OpenAI's for openai, localhost:1234 for lmstudio"
		},
		choices: func(query string) []choice {
			cs := []choice{
				{value: "", title: "Default", note: "OpenAI's for openai:, LM Studio's for lmstudio:"},
				{value: "https://api.openai.com/v1", title: "OpenAI", note: "web search works here"},
				{value: "http://localhost:1234/v1", title: "LM Studio", note: "local"},
				{value: "https://openrouter.ai/api/v1", title: "OpenRouter", note: "many models, one key"},
				{value: "https://api.groq.com/openai/v1", title: "Groq", note: "fast hosted models"},
			}
			if q := strings.TrimSpace(query); strings.HasPrefix(q, "http") {
				cs = append([]choice{{value: q, title: q, note: "what you typed"}}, cs...)
			}
			return cs
		},
	},
	{
		key: settings.AIKey, title: "API key", icon: "󰌆",
		explain: "The key for the API URL. Type it in the bar and press Enter on the row\nthat repeats it. It is kept in the settings file, mode 600, yours\nalone; OPENAI_API_KEY in the environment is used when this is empty.",
		value: func() string {
			if v := settings.Get(settings.AIKey, ""); v != "" {
				return "set (" + strconv.Itoa(len(v)) + " characters)"
			}
			return "not set"
		},
		choices: func(query string) []choice {
			cs := []choice{{value: "", title: "None", note: "use OPENAI_API_KEY from the environment, if any"}}
			if q := strings.TrimSpace(query); len(q) >= 8 {
				cs = append([]choice{{value: q, title: "Use what you typed", note: strconv.Itoa(len(q)) + " characters"}}, cs...)
			}
			return cs
		},
	},
	{
		key: settings.Recent, title: "Used recently", icon: "󰔟",
		explain: "Whether the list starts with the five things you opened most recently,\nmarked \"recent\". Stats, a row at the root, shows the whole log; its\nctrl-k menu clears it.",
		value:   func() string { return settings.Get(settings.Recent, settings.RecentDefault) },
		choices: func(string) []choice {
			return []choice{{value: "on", title: "On", note: "the five most recent first"}, {value: "off", title: "Off", note: "the list in its own order"}}
		},
	},
	{
		key: settings.Render, title: "Transcript renderer", icon: "󰉿",
		explain: "What draws an AI answer in the pane. Built in: swoop's own markdown\nrenderer, no process. Or any command: the answer on its stdin, its\nstdout shown; glow, for one.",
		value: func() string {
			if r := settings.Get(settings.Render, ""); r != "" {
				return r
			}
			return "built in"
		},
		choices: func(string) []choice {
			return []choice{
				{value: "", title: "Built in", note: "swoop's own renderer"},
				{value: "glow -s dark", title: "glow", note: "needs glow on PATH"},
				{value: "swoop-md", title: "swoop-md", note: "the built-in one as a command"},
			}
		},
	},
}

// The row that opens the config folder: not a setting, a door.
const folderID = "folder"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	arg := func(i int) string {
		if len(os.Args) > i {
			return os.Args[i]
		}
		return ""
	}
	var err error
	switch os.Args[1] {
	case "list":
		err = protocol.Write(os.Stdout, []protocol.Item{{ID: "settings", Kind: "view", Icon: "", Title: "Settings", Subtitle: "hotkey, AI model, preview width, web search"}})
	case "view":
		err = view(arg(2), strings.TrimSpace(arg(3)))
	case "preview":
		err = preview(arg(2))
	case "run":
		err = run(arg(2))
	case "actions":
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-settings:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: swoop-settings list | view settings|<key> [query] | preview <id> | run <id>")
	os.Exit(2)
}

func find(key string) *setting {
	for i := range all {
		if all[i].key == key {
			return &all[i]
		}
	}
	return nil
}

// view prints the pane's rows: the settings, or one setting's choices.
// The query filters the settings by title; for choices it may also be a
// value of the user's own, which the setting decides.
func view(id, query string) error {
	if id == "settings" {
		var items []protocol.Item
		for _, s := range all {
			if query != "" && !strings.Contains(strings.ToLower(s.title), strings.ToLower(query)) {
				continue
			}
			items = append(items, protocol.Item{ID: s.key, Kind: "view", Icon: s.icon, Title: s.title, Subtitle: s.value()})
		}
		if query == "" || strings.Contains("open the config folder", strings.ToLower(query)) {
			items = append(items, protocol.Item{ID: folderID, Kind: "command", Icon: "", Title: "Open the config folder", Subtitle: settings.Dir()})
		}
		return protocol.Write(os.Stdout, items)
	}
	s := find(id)
	if s == nil {
		return fmt.Errorf("no setting %q", id)
	}
	current := settings.Get(s.key, "")
	var items []protocol.Item
	for _, c := range s.choices(query) {
		// The pane's own matching is off inside a view, so the choices
		// filter on what was typed themselves; a row the setting made
		// from the typed text is always shown.
		if query != "" && c.note != "what you typed" && !strings.Contains(strings.ToLower(c.title), strings.ToLower(query)) {
			continue
		}
		note := c.note
		if c.value == current || (current == "" && c.value == "" && s.key != settings.AI) {
			note = strings.TrimSpace("current  " + note)
		}
		// Kind refresh: Enter sets the value and returns to the Settings
		// pane, reloaded, so the subtitle shows the new value.
		items = append(items, protocol.Item{ID: s.key + "=" + c.value, Kind: "refresh", Icon: "", Title: c.title, Subtitle: note})
	}
	return protocol.Write(os.Stdout, items)
}

func preview(id string) error {
	key, _, _ := strings.Cut(id, "=")
	if key == folderID {
		fmt.Println("The folder with swoop's files: config, the AI line, the clipboard\nignore list, shell-mac.json, and a place for extensions of your own.")
		return nil
	}
	if key == "settings" {
		fmt.Println("Every setting is one line in\n" + settings.Path() + "\n\nEnter on a row to change it. Font size is cmd+plus and cmd+minus in\nthe frame; the panel's size is set by dragging an edge.")
		return nil
	}
	s := find(key)
	if s == nil {
		return nil
	}
	fmt.Printf("\x1b[1m%s\x1b[22m  %s\n\n%s\n\n\x1b[2m%s = … in %s\x1b[22m\n", s.title, s.value(), s.explain, s.key, settings.Path())
	return nil
}

// run sets a value, or opens the folder. An id with no "=" is a row that
// has nothing to run (the settings themselves are views).
func run(id string) error {
	if id == folderID {
		return openFolder(settings.Dir())
	}
	key, value, ok := strings.Cut(id, "=")
	if !ok {
		return nil
	}
	if find(key) == nil {
		return fmt.Errorf("no setting %q", key)
	}
	if key == settings.Preview {
		value = strconv.Itoa(settings.ClampPreview(atoi(value, settings.PreviewDefault)))
	}
	return settings.Set(key, value)
}

func openFolder(dir string) error {
	_ = os.MkdirAll(dir, 0o700)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", filepath.FromSlash(dir))
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Run()
}

// hotkeyChoices are the keys most launchers use, and whatever was typed
// when it looks like one: "ctrl+alt+k" in the bar becomes a row.
func hotkeyChoices(query string) []choice {
	cs := []choice{
		{value: "alt+shift+space", title: "alt+shift+space", note: "the default"},
		{value: "alt+space", title: "alt+space", note: "Ghostty's quick terminal uses this"},
		{value: "ctrl+space", title: "ctrl+space", note: ""},
		{value: "ctrl+alt+space", title: "ctrl+alt+space", note: ""},
		{value: "cmd+shift+space", title: "cmd+shift+space", note: "the leading launcher's default"},
		{value: "cmd+ctrl+space", title: "cmd+ctrl+space", note: ""},
	}
	q := strings.ToLower(strings.ReplaceAll(query, " ", ""))
	if strings.Contains(q, "+") && looksLikeHotkey(q) {
		for _, c := range cs {
			if c.value == q {
				return cs
			}
		}
		cs = append([]choice{{value: q, title: q, note: "what you typed"}}, cs...)
	}
	return cs
}

// looksLikeHotkey accepts mod+mod+key with the modifiers the frame knows
// and a key that is one character or a name it has, without pretending to
// be the frame's parser: a bad one is refused by the frame and logged.
func looksLikeHotkey(spec string) bool {
	parts := strings.Split(spec, "+")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts[:len(parts)-1] {
		switch p {
		case "cmd", "super", "alt", "option", "ctrl", "control", "shift":
		default:
			return false
		}
	}
	return parts[len(parts)-1] != ""
}

func aiValue() string { return models.Current() }

func aiChoices(query string) []choice {
	var cs []choice
	for _, m := range models.Choices(query) {
		cs = append(cs, choice{value: m.Value, title: m.Title, note: m.Note})
	}
	return cs
}

func atoi(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}
