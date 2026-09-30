// swoop-settings is the Settings pane: the extension behind the Settings
// row and cmd+, in the frame. It speaks the extension contract, so the
// launcher routes to it like any other.
//
//	swoop-settings list                 the Settings row
//	swoop-settings view settings [q]    one row per setting, the value in the subtitle
//	swoop-settings view extensions [q]  one row per extension, on or off
//	swoop-settings view <key> [q]       the choices for one setting
//	swoop-settings preview <id>         what the setting does and where it lives
//	swoop-settings run <id>             <key>=<value> sets it; extension/<name> turns it
//	                                    on or off; folder opens the config folder
//	swoop-settings edit <file>          the editor setting, run on file, for a script
//	                                    extension that has no Go to call it with
//
// Every value is one line in ~/.config/swoop/config, the file
// internal/settings reads and writes. The frame watches that file for the
// hotkey, so a change here takes effect with no restart.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/match"
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
	// words are extra words the filter matches, never shown: what a person
	// types for this row that neither its title nor its value holds.
	words   string
	value   func() string // the subtitle: the current value, in words
	choices func(query string) []choice
}

type choice struct {
	value string // what goes in the file
	title string
	note  string
	// typed marks a row the setting made from the text in the bar. The
	// pane always shows it: its title need not repeat that text, and its
	// note is free to say anything (#160).
	typed bool
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
		key: settings.AIURL, title: "API URL", icon: "󰖟", words: "ai",
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
				cs = append([]choice{{value: q, title: q, note: "what you typed", typed: true}}, cs...)
			}
			return cs
		},
	},
	{
		key: settings.AIKey, title: "API key", icon: "󰌆", words: "ai",
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
				cs = append([]choice{{value: q, title: "Use what you typed", note: strconv.Itoa(len(q)) + " characters", typed: true}}, cs...)
			}
			return cs
		},
	},
	{
		key: settings.EditorKey, title: "Editor", icon: "󰏫",
		explain: "What a note opens in, and the task list, and a text file from Search\nFiles: inside the panel, with the whole terminal, and quitting it brings\nthe list back. Empty means $EDITOR, and with neither, nano. Type a\ncommand to use it: emacs -nw, or anything on your PATH.",
		value:   editorValue,
		choices: editorChoices,
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
		key: settings.Week, title: "Week starts on", icon: "󰃭",
		explain: "The first day of the calendar week. Tasks and Reminders group what is\ndue under This week up to the last day of it.",
		value: func() string {
			if settings.WeekStart() == time.Sunday {
				return "Sunday"
			}
			return "Monday"
		},
		choices: func(string) []choice {
			return []choice{{value: "monday", title: "Monday", note: "the week ends on Sunday"}, {value: "sunday", title: "Sunday", note: "the week ends on Saturday"}}
		},
	},
	{
		key: settings.Skin, title: "Emoji skin tone", icon: "󱠫",
		explain: "The skin tone Enter pastes, for an emoji that takes one. ctrl-k on an\nemoji offers all six for one use.",
		value: func() string {
			n := atoi(settings.Get(settings.Skin, settings.SkinDefault), 0)
			if n < 0 || n >= len(skinTones) {
				n = 0
			}
			return skinTones[n].title
		},
		choices: func(string) []choice { return skinTones },
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

// skinTones are the six emoji tones, 0 for none and 1 to 5 from light
// to dark: the values swoop-emoji reads.
var skinTones = []choice{
	{value: "0", title: "✋ None", note: "the yellow of the emoji as drawn"},
	{value: "1", title: "✋🏻 Light", note: ""},
	{value: "2", title: "✋🏼 Medium-light", note: ""},
	{value: "3", title: "✋🏽 Medium", note: ""},
	{value: "4", title: "✋🏾 Medium-dark", note: ""},
	{value: "5", title: "✋🏿 Dark", note: ""},
}

// The row that opens the config folder: not a setting, a door.
const folderID = "folder"

// root is the Settings row at the root.
var root = protocol.Item{ID: "settings", Kind: "view", Icon: "󰒓", Title: "Settings", Subtitle: "hotkey, AI model, editor, preview width, extensions"}

// Every row has an icon exactly one cell wide, the width of the Nerd Font
// glyphs the settings use. The launcher shows "icon title", so an empty
// icon, or an emoji two cells wide, moves that row's title out of the
// column. blank is the icon for a row with nothing to show.
const blank = " "

// The Extensions row's view, and the prefix of the ids of its rows, one
// per extension: "extension/reminders".
const (
	extensionsID    = "extensions"
	extensionPrefix = "extension/"
)

// The marks for on and off: the ticked and empty boxes Tasks uses.
const (
	iconOn  = "󰄲"
	iconOff = "󰄱"
)

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
		err = protocol.Write(os.Stdout, []protocol.Item{root})
	case "view":
		err = view(arg(2), strings.TrimSpace(arg(3)))
	case "preview":
		err = preview(arg(2))
	case "run":
		err = run(arg(2))
	case "edit":
		if arg(2) == "" {
			usage()
		}
		err = settings.Edit(arg(2))
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
	fmt.Fprintln(os.Stderr, "usage: swoop-settings list | view settings|extensions|<key> [query] | preview <id> | run <id> | edit <file>")
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

// view prints the pane's rows.
func view(id, query string) error {
	items, err := rows(id, query)
	if err != nil {
		return err
	}
	return protocol.Write(os.Stdout, items)
}

// rows are the pane's rows: the settings, the extensions, or one
// setting's choices. The query filters each of them by words, in the
// title or the subtitle, and the best match comes first (internal/match):
// "key api" and "akey" find API key, and "nvim" finds the Editor row
// whose value is nvim (#196, #201). For choices the query may also be a
// value of the user's own, which the setting decides.
func rows(id, query string) ([]protocol.Item, error) {
	q := match.New(query)
	var best match.Best[protocol.Item]
	if id == "settings" {
		for _, s := range all {
			value := s.value()
			if score, ok := q.Score(s.title, value, s.words); ok {
				best.Add(protocol.Item{ID: s.key, Kind: "view", Icon: s.icon, Title: s.title, Subtitle: value}, score)
			}
		}
		value := extensionsValue()
		if score, ok := q.Score("Extensions", value); ok {
			best.Add(protocol.Item{ID: extensionsID, Kind: "view", Icon: "󰏗", Title: "Extensions", Subtitle: value}, score)
		}
		// The folder row matches on its title alone. Its subtitle is a
		// path, not a value, and the letters of a home folder's name would
		// bring the row up for words that have nothing to do with it.
		if score, ok := q.Score("Open the config folder"); ok {
			best.Add(protocol.Item{ID: folderID, Kind: "command", Icon: "󰉋", Title: "Open the config folder", Subtitle: settings.Dir()}, score)
		}
		return best.Rows(), nil
	}
	if id == extensionsID {
		return extensionRows(query), nil
	}
	s := find(id)
	if s == nil {
		return nil, fmt.Errorf("no setting %q", id)
	}
	current := settings.Get(s.key, "")
	for _, c := range s.choices(query) {
		note := c.note
		if c.value == current || (current == "" && c.value == "" && s.key != settings.AI) {
			note = strings.TrimSpace("current  " + note)
		}
		// The pane's own matching is off inside a view, so the choices
		// filter on what was typed themselves, by words in the title or
		// the note, best first; a row the setting made from the typed
		// text is always shown, where the setting put it.
		// Kind refresh: Enter sets the value and returns to the Settings
		// pane, reloaded, so the subtitle shows the new value.
		it := protocol.Item{ID: s.key + "=" + c.value, Kind: "refresh", Icon: blank, Title: c.title, Subtitle: note}
		if c.typed {
			best.Pin(it)
		} else if score, ok := q.Score(c.title, note); ok {
			best.Add(it, score)
		}
	}
	return best.Rows(), nil
}

func preview(id string) error {
	if name, ok := strings.CutPrefix(id, extensionPrefix); ok {
		return previewExtension(name)
	}
	key, _, _ := strings.Cut(id, "=")
	if key == folderID {
		fmt.Println("The folder with swoop's files: config, the AI line, the clipboard\nignore list, shell-mac.json, and a place for extensions of your own.")
		return nil
	}
	if key == extensionsID {
		fmt.Println("Every extension swoop found, bundled and your own. Enter turns one on\nor off; one that is off has no rows at the root, no pane, no actions.\nSettings stays on: it is the way back.\n\n\x1b[2m" + settings.Off + " = … in " + settings.Path() + "\x1b[22m")
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
	if name, ok := strings.CutPrefix(id, extensionPrefix); ok {
		return flip(name)
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
		cs = append([]choice{{value: q, title: q, note: "what you typed", typed: true}}, cs...)
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

// editors are the Editor row's choices, each shown only when its first
// word is on PATH: a choice that cannot run is a trap.
var editors = []choice{
	{value: "nvim", title: "nvim", note: ""},
	{value: "vim", title: "vim", note: ""},
	{value: "nano", title: "nano", note: ""},
	{value: "pico", title: "pico", note: ""},
	{value: "hx", title: "hx", note: "Helix"},
	{value: "micro", title: "micro", note: ""},
	{value: "emacs -nw", title: "emacs -nw", note: "in the terminal, not its own window"},
}

// editorValue is the Editor row's subtitle: what will run, and why.
func editorValue() string {
	if v := settings.Get(settings.EditorKey, ""); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("EDITOR")); v != "" {
		return v + " · from $EDITOR"
	}
	return settings.EditorDefault + " · the default"
}

// editorChoices are Default, the editors found on PATH, and what was
// typed when its first word is a program on PATH, so "nvim --clean"
// becomes a row.
func editorChoices(query string) []choice {
	cs := []choice{{value: "", title: "Default", note: "$EDITOR, else " + settings.EditorDefault}}
	onPath := func(line string) bool {
		f := strings.Fields(line)
		if len(f) == 0 {
			return false
		}
		_, err := exec.LookPath(f[0])
		return err == nil
	}
	typed := strings.Join(strings.Fields(query), " ")
	for _, c := range editors {
		if onPath(c.value) {
			cs = append(cs, c)
		}
		if c.value == typed {
			typed = ""
		}
	}
	if typed != "" && onPath(typed) {
		cs = append([]choice{{value: typed, title: typed, note: "what you typed", typed: true}}, cs...)
	}
	return cs
}

func aiValue() string { return models.Current() }

func aiChoices(query string) []choice {
	var cs []choice
	for _, m := range models.Choices(query) {
		cs = append(cs, choice{value: m.Value, title: m.Title, note: m.Note, typed: m.Typed})
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

// extensionsValue is the Extensions row's subtitle: how many are on.
func extensionsValue() string {
	all := ext.DiscoverAll(ext.Dirs())
	on := len(ext.Discover(ext.Dirs()))
	if on == len(all) {
		return fmt.Sprintf("all %d on", on)
	}
	return fmt.Sprintf("%d on, %d off", on, len(all)-on)
}

// extensionRows is the Extensions view: every extension found, on or off,
// by name, filtered on what was typed by words in the name or the
// subtitle, best first. Each row is kind toggle:
// Enter flips it and the view stays, reloaded, with the new mark.
func extensionRows(query string) []protocol.Item {
	off := settings.OffList()
	q := match.New(query)
	var best match.Best[protocol.Item]
	for _, e := range ext.DiscoverAll(ext.Dirs()) {
		it := protocol.Item{ID: extensionPrefix + e.Name, Kind: "toggle", Icon: iconOn, Title: e.Name, Subtitle: purpose(e.Name, e.Exe)}
		switch {
		case e.Name == settings.AlwaysOn:
			it.Subtitle = "always on · " + it.Subtitle
		case slices.Contains(off, e.Name):
			// The box says it, and the word does too, for a font without
			// the glyph.
			it.Icon = iconOff
			it.Subtitle = strings.TrimSuffix("off · "+it.Subtitle, " · ")
		}
		if score, ok := q.Score(it.Title, it.Subtitle); ok {
			best.Add(it, score)
		}
	}
	return best.Rows()
}

// flip turns one extension on if it is off, and off if it is on. Settings
// stays on whatever is asked: ParseOff drops it.
func flip(name string) error {
	off := settings.OffList()
	if i := slices.Index(off, name); i >= 0 {
		off = slices.Delete(off, i, i+1)
	} else {
		off = append(off, name)
	}
	return settings.Set(settings.Off, settings.FormatOff(off))
}

func previewExtension(name string) error {
	for _, e := range ext.DiscoverAll(ext.Dirs()) {
		if e.Name != name {
			continue
		}
		state := "on"
		if slices.Contains(settings.OffList(), name) {
			state = "off"
		}
		if name == settings.AlwaysOn {
			state = "always on"
		}
		fmt.Printf("\x1b[1m%s\x1b[22m  %s\n\n%s\n\n\x1b[2m%s\n%s = … in %s\x1b[22m\n", e.Name, state, purpose(e.Name, e.Exe), e.Dir, settings.Off, settings.Path())
		return nil
	}
	return nil
}

// purpose is an extension's one line: the first sentence of the first
// comment in its script, without the "name:" it opens with. A script
// without one, or a program that is not a script, has none.
func purpose(name, exe string) string {
	f, err := os.Open(exe)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() || !strings.HasPrefix(sc.Text(), "#!") || !sc.Scan() {
		return ""
	}
	line, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "#")
	if !ok {
		return ""
	}
	line = strings.TrimPrefix(strings.TrimSpace(line), name+": ")
	if i := strings.Index(line, ". "); i >= 0 {
		line = line[:i]
	}
	// The line may stop mid-sentence, where the comment wraps.
	return strings.TrimRight(line, ".,;: ")
}
