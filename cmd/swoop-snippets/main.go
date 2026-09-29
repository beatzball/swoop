// swoop-snippets is the snippets extension: named text, each snippet a
// row at the root. Enter fills the placeholders, pastes the text into
// the app in front (see internal/paste), and leaves the caret at
// {cursor}. The snippets are files, one per snippet, in
// ~/.config/swoop/snippets/, so any editor is the editor and git is the
// sync. See store.go for the file and fill.go for the placeholders.
//
//	swoop-snippets list                  the root rows, one per snippet
//	swoop-snippets preview <file>        the text, placeholders filled
//	swoop-snippets actions <file>        Paste, Copy, Delete
//	swoop-snippets run <file> [action]   paste, copy, or delete
//	swoop-snippets add <name> [keyword]  a snippet from stdin
//	swoop-snippets import <export.json>
//
// An id is the snippet's file name in the folder.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/beatzball/swoop/internal/settings"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
)

// icon is the rows' glyph, a clipboard with a page on it (nf-fa-paste).
const icon = "\uf0ea"

// viewID is the Snippets row, always at the root, so the feature can be
// found whatever the snippets are called. Its pane lists newID first,
// then every snippet. A plus, so no file can be called the same.
const (
	viewID = "+snippets"
	newID  = "+new"
)

func main() {
	if len(os.Args) < 2 {
		usageExit()
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
		err = list()
	case "view":
		err = view(arg(3))
	case "preview":
		err = preview(arg(2))
	case "actions":
		err = actions(arg(2))
	case "run":
		err = run(arg(2), arg(3))
	case "add":
		if strings.TrimSpace(arg(2)) == "" {
			usageExit()
		}
		err = add(arg(2), arg(3))
	case "import":
		if arg(2) == "" {
			usageExit()
		}
		err = importJSON(arg(2))
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-snippets:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-snippets list | view snippets [text] | preview <file> | actions <file> | run <file> [paste|copy|delete] | add <name> [keyword] < text | import <export.json>")
	os.Exit(2)
}

// list prints the Snippets row, then one row per snippet with the
// keyword in the subtitle. fzf filters at the root, so every snippet is
// printed whatever was typed.
func list() error {
	snippets, err := heads()
	if err != nil {
		return err
	}
	items := make([]protocol.Item, 0, len(snippets)+1)
	items = append(items, protocol.Item{ID: viewID, Kind: "view", Icon: icon, Title: "Snippets", Subtitle: "new one, or all of them"})
	return protocol.Write(os.Stdout, append(items, rows(snippets)...))
}

// view prints the Snippets pane: New snippet first, then the snippets
// whose name or keyword holds every word typed. The extension filters
// here; fzf's matching is off inside a pane.
func view(query string) error {
	snippets, err := heads()
	if err != nil {
		return err
	}
	words := strings.Fields(strings.ToLower(query))
	var kept []Snippet
	for _, s := range snippets {
		hay := strings.ToLower(s.Name + " " + s.Keyword)
		all := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				all = false
				break
			}
		}
		if all {
			kept = append(kept, s)
		}
	}
	sub := "a new file, opened in the editor"
	if len(snippets) == 0 {
		sub = "none yet: Enter makes the Signature example in the editor"
	}
	items := []protocol.Item{{ID: newID, Kind: "terminal", Icon: icon, Title: "New snippet", Subtitle: sub}}
	return protocol.Write(os.Stdout, append(items, rows(kept)...))
}

// rows is one row per snippet.
func rows(snippets []Snippet) []protocol.Item {
	items := make([]protocol.Item, len(snippets))
	for i, s := range snippets {
		sub := "snippet"
		if s.Keyword != "" {
			sub = s.Keyword + " · snippet"
		}
		items[i] = protocol.Item{ID: s.File, Kind: "snippet", Icon: icon, Title: s.Name, Subtitle: sub}
	}
	return items
}

// howTo is the preview of the Snippets and New snippet rows.
func howTo() string {
	return fmt.Sprintf("A snippet is a file in %s:\n\n"+
		"  Signature\n  keyword: ;sig\n\n  Best,\n  {cursor}\n\n"+
		"The first line is the name, the keyword line is optional, the rest is\n"+
		"the text. Enter pastes it into the app you came from.\n\n"+
		"New snippet makes this example the first time, in your editor;\n"+
		"after that, a file named by the date and time.\n\n"+
		"Placeholders: {date} {time} {clipboard} {uuid}, and {cursor} for\n"+
		"where the caret ends up.\n\n"+
		"Or: swoop-snippets import export.json (name, text, keyword)\n", tilde(dir()))
}

func preview(id string) error {
	if id == viewID || id == newID {
		fmt.Print(howTo())
		return nil
	}
	s, err := read(id)
	if err != nil {
		return err
	}
	text, _ := Fill(s.Text, realEnv(paste.Clipboard))
	fmt.Printf("\x1b[1m%s\x1b[22m\n", s.Name)
	if s.Keyword != "" {
		fmt.Printf("keyword %s\n", s.Keyword)
	}
	fmt.Printf("\n%s\n\n\x1b[2m%s\x1b[22m\n", text, tilde(dir()+"/"+s.File))
	return nil
}

// tilde shows the home directory as ~, so no home path reaches the
// screen.
func tilde(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// actions is the ctrl-k menu. Copy fills the placeholders too; the
// caret has nowhere to go, so {cursor} is only removed. Delete is a
// refresh: the list comes back without the snippet.
func actions(id string) error {
	if id == viewID || id == newID {
		return nil
	}
	if _, err := read(id); err != nil {
		return err
	}
	return protocol.Write(os.Stdout, []protocol.Item{
		{ID: "paste", Kind: "action", Icon: icon, Title: "Paste", Subtitle: "Enter"},
		{ID: "copy", Kind: "action", Icon: icon, Title: "Copy", Subtitle: "to the clipboard"},
		{ID: "new", Kind: "terminal", Icon: icon, Title: "New snippet", Subtitle: "a new file, opened in the editor"},
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: "Remove the snippet's file"},
	})
}

func run(id, action string) error {
	if id == viewID {
		return nil
	}
	if id == newID || action == "new" {
		have, err := heads()
		if err != nil {
			return err
		}
		if len(have) == 0 {
			// The first snippet: the example from the how-to, opened in
			// the editor here in the panel. Quit, and it is a row.
			return edit(Snippet{Name: "Signature", Keyword: ";sig", Text: "Best,\n{cursor}"})
		}
		// A file named by the date and time, like a New note with
		// nothing typed; the first line is the name, so the editor is
		// where it gets a real one.
		return edit(Snippet{Name: time.Now().Format("2006-01-02 15:04")})
	}
	s, err := read(id)
	if err != nil {
		return err
	}
	text, back := Fill(s.Text, realEnv(paste.Clipboard))
	switch action {
	case "", "open", "paste":
	case "copy":
		return paste.Copy(text)
	case "delete":
		return remove(id)
	default:
		return fmt.Errorf("no action %q for a snippet", action)
	}
	note, err := paste.PasteBack(text, back)
	if err != nil {
		return err
	}
	if note != "" {
		paste.Tell(note)
	}
	return nil
}

// edit writes s as a file, unless a snippet of that name is already
// there (put replaces by name, and an edited example must stay edited),
// opens it in the editor, and asks the launcher to come back on its row
// (see nav.Back, SWOOP_LAND).
func edit(s Snippet) error {
	find := func() (string, error) {
		have, err := heads()
		if err != nil {
			return "", err
		}
		for _, h := range have {
			if h.Name == oneLine(s.Name) {
				return h.File, nil
			}
		}
		return "", nil
	}
	file, err := find()
	if err != nil {
		return err
	}
	if file == "" {
		if err := put([]Snippet{s}); err != nil {
			return err
		}
		if file, err = find(); err != nil {
			return err
		}
	}
	if file == "" {
		return errors.New("the snippet was not written")
	}
	if land := os.Getenv("SWOOP_LAND"); land != "" {
		if err := os.WriteFile(land, []byte(file+"\n"), 0o600); err != nil {
			return err
		}
	}
	return settings.Edit(filepath.Join(dir(), file))
}

// add writes one snippet, its text read from stdin.
func add(name, keyword string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	return put([]Snippet{{Name: name, Keyword: keyword, Text: strings.TrimSuffix(string(data), "\n")}})
}

// exportedSnippet is one entry of a snippets export, the JSON array of
// objects with name, text and keyword that other launchers write.
type exportedSnippet struct {
	Name    string `json:"name"`
	Text    string `json:"text"`
	Keyword string `json:"keyword"`
}

// importJSON reads an export and writes a file for every snippet in it,
// replacing a snippet of the same name. It prints how many it took.
func importJSON(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var entries []exportedSnippet
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("%s: not a snippets export: %w", file, err)
	}
	var snippets []Snippet
	for _, e := range entries {
		if strings.TrimSpace(e.Name) == "" || e.Text == "" {
			continue
		}
		snippets = append(snippets, Snippet{Name: e.Name, Keyword: e.Keyword, Text: e.Text})
	}
	if len(snippets) == 0 {
		return errors.New("no snippets in " + file)
	}
	if err := put(snippets); err != nil {
		return err
	}
	fmt.Printf("imported %d snippets into %s\n", len(snippets), tilde(dir()))
	return nil
}
