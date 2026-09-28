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
	"io"
	"os"
	"strings"

	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
)

// icon is the rows' glyph, a clipboard with a page on it (nf-fa-paste).
const icon = "\uf0ea"

// helpID is the one row shown while there are no snippets, so the
// feature can be found and says how to start.
const helpID = "+help"

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
	fmt.Fprintln(os.Stderr, "usage: swoop-snippets list | preview <file> | actions <file> | run <file> [paste|copy|delete] | add <name> [keyword] < text | import <export.json>")
	os.Exit(2)
}

// list prints one row per snippet, the keyword in the subtitle. fzf
// filters at the root, so every snippet is printed whatever was typed.
func list() error {
	snippets, err := heads()
	if err != nil {
		return err
	}
	if len(snippets) == 0 {
		return protocol.Write(os.Stdout, []protocol.Item{{ID: helpID, Kind: "snippet", Icon: icon, Title: "Snippets", Subtitle: "none yet: Enter for how to add one"}})
	}
	items := make([]protocol.Item, len(snippets))
	for i, s := range snippets {
		sub := "snippet"
		if s.Keyword != "" {
			sub = s.Keyword + " · snippet"
		}
		items[i] = protocol.Item{ID: s.File, Kind: "snippet", Icon: icon, Title: s.Name, Subtitle: sub}
	}
	return protocol.Write(os.Stdout, items)
}

// howTo is the preview of the help row, and what Enter on it says.
func howTo() string {
	return fmt.Sprintf("A snippet is a file in %s:\n\n"+
		"  Signature\n  keyword: ;sig\n\n  Best,\n  {cursor}\n\n"+
		"The first line is the name, the keyword line is optional, the rest is\n"+
		"the text. Enter pastes it into the app you came from.\n\n"+
		"Placeholders: {date} {time} {clipboard} {uuid}, and {cursor} for\n"+
		"where the caret ends up.\n\n"+
		"Or: swoop-snippets import export.json (name, text, keyword)\n", tilde(dir()))
}

func preview(id string) error {
	if id == helpID {
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
	if id == helpID {
		return nil
	}
	if _, err := read(id); err != nil {
		return err
	}
	return protocol.Write(os.Stdout, []protocol.Item{
		{ID: "paste", Kind: "action", Icon: icon, Title: "Paste", Subtitle: "Enter"},
		{ID: "copy", Kind: "action", Icon: icon, Title: "Copy", Subtitle: "to the clipboard"},
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: "Remove the snippet's file"},
	})
}

func run(id, action string) error {
	if id == helpID {
		if err := os.MkdirAll(dir(), 0o700); err != nil {
			return err
		}
		paste.Tell(howTo())
		return nil
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
