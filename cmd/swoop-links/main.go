// swoop-links is the quicklinks extension: named links, each one a row at
// the root. A link is a URL, a path, or a deeplink, with an app to open
// it with if the default is not wanted. A link that holds {argument} is
// a pane: Enter asks for text, and Enter again opens the link with the
// text in its place. The links are one file, ~/.config/<name>/quicklinks.tsv,
// edited by hand, by `add`, or by `import` of a JSON export.
//
//	swoop-links list [query]         the root rows, one per link
//	swoop-links view <name> [text]   the pane of a link with {argument}
//	swoop-links preview <id>         the link's facts
//	swoop-links actions <id>         Copy Link, Delete
//	swoop-links run <id> [action]    open, copy, or delete
//	swoop-links add <name> <link> [app]
//	swoop-links import <export.json>
//	swoop-links defaults             print the links that ship
//
// An id is the link's name; inside a pane it is the name, a unit
// separator, and the text typed, so one id names both the link and what
// to put in it.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/beatzball/swoop/internal/protocol"
)

// sep joins a name and an argument into one id. A control character, so
// no name typed by a person can hold it.
const sep = "\x1f"

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
		err = view(arg(2), arg(3))
	case "preview":
		err = preview(arg(2))
	case "actions":
		err = actions(arg(2))
	case "run":
		err = run(arg(2), arg(3))
	case "add":
		if arg(2) == "" || arg(3) == "" {
			usageExit()
		}
		err = add(Link{Name: arg(2), Link: arg(3), App: arg(4)})
	case "import":
		if arg(2) == "" {
			usageExit()
		}
		err = importJSON(arg(2))
	case "defaults":
		_, err = os.Stdout.WriteString(header() + defaults)
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-links:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-links list | view <name> [text] | preview <id> | actions <id> | run <id> [action] | add <name> <link> [app] | import <export.json> | defaults")
	os.Exit(2)
}

// list prints one row per link. fzf filters at the root, so every link is
// printed whatever was typed. A link with {argument} is a view; the rest
// open on Enter.
func list() error {
	links, err := load()
	if err != nil {
		return err
	}
	var items []protocol.Item
	for _, l := range links {
		it := protocol.Item{ID: l.Name, Kind: "link", Icon: l.Icon(), Title: l.Name, Subtitle: l.Where()}
		if l.HasArgument() {
			it.Kind = "view"
			it.Subtitle += " · type, then Enter"
		}
		items = append(items, it)
	}
	return protocol.Write(os.Stdout, items)
}

// view is the pane of one link: a single row that carries what was typed.
// With nothing typed yet the row says what to do, and Enter on it opens
// the link with the placeholder empty, which for a search is its front
// page. The subtitle is the link as it will be opened, so what Enter does
// is never a surprise.
func view(name, text string) error {
	links, err := load()
	if err != nil {
		return err
	}
	l, ok := find(links, name)
	if !ok {
		return fmt.Errorf("no quicklink called %q", name)
	}
	it := protocol.Item{ID: name + sep + text, Kind: "link", Icon: l.Icon(), Title: text, Subtitle: l.Fill(text)}
	if strings.TrimSpace(text) == "" {
		it.Title = "Type, then Enter"
	}
	return protocol.Write(os.Stdout, []protocol.Item{it})
}

// resolve splits an id into its link and, inside a pane, the text.
func resolve(id string) (Link, string, bool, error) {
	name, text, inPane := strings.Cut(id, sep)
	links, err := load()
	if err != nil {
		return Link{}, "", false, err
	}
	l, ok := find(links, name)
	if !ok {
		return Link{}, "", false, fmt.Errorf("no quicklink called %q", name)
	}
	return l, text, inPane, nil
}

// target is what an id opens: the link filled in, or as written.
func target(l Link, text string, inPane bool) string {
	if inPane {
		return l.Fill(text)
	}
	return l.Link
}

func preview(id string) error {
	l, text, inPane, err := resolve(id)
	if err != nil {
		return err
	}
	fmt.Println(l.Name)
	fmt.Println()
	fmt.Println(target(l, text, inPane))
	if l.App != "" {
		fmt.Println()
		fmt.Println("Opens with", l.App)
	}
	if l.HasArgument() && !inPane {
		fmt.Println()
		fmt.Println("Enter, then type what goes in {argument}.")
	}
	fmt.Println()
	fmt.Println("Quicklinks live in", tilde(path()))
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

// actions is the ctrl-k menu. Copy takes the link as it would open, so a
// search typed in a pane copies the whole URL. Delete is a refresh: the
// list comes back without the link.
func actions(id string) error {
	if _, _, _, err := resolve(id); err != nil {
		return err
	}
	return protocol.Write(os.Stdout, []protocol.Item{
		{ID: "open", Kind: "action", Icon: "", Title: "Open", Subtitle: "Enter"},
		{ID: "copy", Kind: "action", Icon: "", Title: "Copy Link", Subtitle: ""},
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: "Remove the quicklink from the file"},
	})
}

func run(id, action string) error {
	l, text, inPane, err := resolve(id)
	if err != nil {
		return err
	}
	switch action {
	case "", "open":
		return openLink(target(l, text, inPane), l.App)
	case "copy":
		return copyText(target(l, text, inPane))
	case "delete":
		return remove(l.Name)
	}
	return fmt.Errorf("no action %q for a quicklink", action)
}

// add appends a link to the user's file. The first add starts the file
// from the defaults, so they stay. A name already there is replaced:
// adding is also how a link is corrected.
func add(l Link) error {
	links, err := load()
	if err != nil {
		return err
	}
	links = upsert(links, l)
	return save(links)
}

func upsert(links []Link, l Link) []Link {
	for i := range links {
		if links[i].Name == l.Name {
			links[i] = l
			return links
		}
	}
	return append(links, l)
}

// remove takes the named link out of the user's file, starting the file
// from the defaults when there is none yet: deleting a default is also
// a thing to remember.
func remove(name string) error {
	links, err := load()
	if err != nil {
		return err
	}
	kept := links[:0]
	for _, l := range links {
		if l.Name != name {
			kept = append(kept, l)
		}
	}
	return save(kept)
}

// exportedLink is one entry of a quicklinks export, the JSON array of
// objects with name, link, openWith, and iconName that other launchers
// write. Only the first three mean anything here; the icon is a glyph
// by scheme.
type exportedLink struct {
	Name     string `json:"name"`
	Link     string `json:"link"`
	OpenWith string `json:"openWith"`
}

// importJSON reads an export and adds every link in it, replacing a
// link of the same name. It prints how many it took.
func importJSON(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var entries []exportedLink
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("%s: not a quicklinks export: %w", file, err)
	}
	links, err := load()
	if err != nil {
		return err
	}
	n := 0
	for _, e := range entries {
		if strings.TrimSpace(e.Name) == "" || strings.TrimSpace(e.Link) == "" {
			continue
		}
		links = upsert(links, Link{Name: e.Name, Link: e.Link, App: e.OpenWith})
		n++
	}
	if n == 0 {
		return errors.New("no links in " + file)
	}
	if err := save(links); err != nil {
		return err
	}
	fmt.Printf("imported %d quicklinks into %s\n", n, tilde(path()))
	return nil
}
