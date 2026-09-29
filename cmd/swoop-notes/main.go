// swoop-notes is the notes extension: markdown files in a folder, one
// root row, Notes, that opens a view of them. The view lists the notes
// newest change first; typing searches titles and text. The preview is
// the note rendered, by the renderer swoop-md uses. Enter opens the file
// in the editor setting, inside the panel: the rows are kind "terminal",
// so quitting the editor brings the view back with the preview redrawn.
// ctrl-k Open in app hands it to the app that owns .md files instead. A
// New note row leads the view and makes a note whose first line is what
// was typed, and opens that; the view comes back with the bar empty and
// the new note under the cursor. See store.go for the folder and the file.
//
//	swoop-notes list                    the root row
//	swoop-notes view notes [query]      New note, then the notes
//	swoop-notes preview <id>            the note, rendered
//	swoop-notes actions <id>            Open in app, Copy, Reveal, Delete
//	swoop-notes run <id> [action]       edit (no action), open, copy, reveal, or delete
//
// An id is the note's file name in the folder, or newID and the text.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/markdown"
	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
)

// icon is the rows' glyph, a sticky note (nf-fa-sticky_note).
const icon = ""

// viewID is the root row, and the one view.
const viewID = "notes"

// newID is the New note row. What was typed follows a slash, so it
// comes back to run in the id: the view's query is not passed to run.
const newID = "+new"

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
		err = protocol.Write(os.Stdout, []protocol.Item{{ID: viewID, Kind: "view", Icon: icon, Title: "Notes", Subtitle: "Search, read and write markdown notes"}})
	case "view":
		err = view(arg(3))
	case "preview":
		err = preview(arg(2))
	case "actions":
		err = actions(arg(2))
	case "run":
		err = run(arg(2), arg(3))
	default:
		usageExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-notes:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-notes list | view notes [query] | preview <id> | actions <id> | run <id> [open|copy|reveal|delete]")
	os.Exit(2)
}

// view prints New note, then the notes that match. Every row is kind
// "terminal": Enter runs the editor with the terminal, and the view
// stays (see internal/nav).
func view(query string) error {
	notes, err := find(dir(), query)
	if err != nil {
		return err
	}
	items := make([]protocol.Item, 0, len(notes)+1)
	title := strings.TrimSpace(query)
	if title == "" {
		items = append(items, protocol.Item{ID: newID, Kind: "terminal", Icon: "", Title: "New note", Subtitle: "type its first line, or Enter for " + stamp()})
	} else {
		items = append(items, protocol.Item{ID: newID + "/" + title, Kind: "terminal", Icon: "", Title: "New note: " + title, Subtitle: "Enter makes it and opens it"})
	}
	for _, n := range notes {
		sub := n.Mod.Format("2006-01-02 15:04")
		if n.Line != "" {
			sub = cut(n.Line, 80)
		}
		items = append(items, protocol.Item{ID: n.File, Kind: "terminal", Icon: icon, Title: n.Title, Subtitle: sub})
	}
	return protocol.Write(os.Stdout, items)
}

// cut shortens s to n characters, with an ellipsis when it was longer.
func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// newTitle says whether id is the New note row, and what was typed.
func newTitle(id string) (string, bool) {
	if id == newID {
		return "", true
	}
	if t, ok := strings.CutPrefix(id, newID+"/"); ok {
		return t, true
	}
	return "", false
}

func preview(id string) error {
	if t, ok := newTitle(id); ok {
		if t == "" {
			fmt.Printf("Type the new note's first line, then Enter.\nWith nothing typed, Enter makes a note titled\n\n  # %s\n\nIt goes in %s, one markdown file per note,\nand opens in %s, here in the panel.\n", stamp(), tilde(dir()), strings.Join(settings.Editor(), " "))
			return nil
		}
		fmt.Printf("Enter makes a note in %s that starts\n\n  # %s\n\nand opens it.\n", tilde(dir()), t)
		return nil
	}
	p, err := path(id)
	if err != nil {
		return err
	}
	text, err := readCapped(p)
	if err != nil {
		return err
	}
	fmt.Print(markdown.Render(text, width()))
	fmt.Printf("\n\x1b[2m%s\x1b[22m\n", tilde(p))
	return nil
}

// width is the preview's width, as fzf says, else 80.
func width() int {
	for _, name := range []string{"FZF_PREVIEW_COLUMNS", "COLUMNS"} {
		if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
			return v
		}
	}
	return 80
}

// tilde shows the home directory as ~, so no home path reaches the
// screen.
func tilde(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// actions is the ctrl-k menu. Enter already edits, so the menu holds the
// rest: the app that owns .md files, for when a window is wanted, and
// Delete, a refresh: the view comes back without the note.
func actions(id string) error {
	if _, ok := newTitle(id); ok {
		return nil
	}
	if _, err := path(id); err != nil {
		return err
	}
	return protocol.Write(os.Stdout, []protocol.Item{
		{ID: "open", Kind: "action", Icon: icon, Title: "Open in app", Subtitle: "the app that opens .md files"},
		{ID: "copy", Kind: "action", Icon: "", Title: "Copy", Subtitle: "the note's text"},
		{ID: "reveal", Kind: "action", Icon: "", Title: "Reveal", Subtitle: "show the file in its folder"},
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: "move it to " + deletedDir + "/"},
	})
}

func run(id, action string) error {
	if t, ok := newTitle(id); ok {
		p, err := create(dir(), t)
		if err != nil {
			return err
		}
		// Come back on the note just made, with the bar cleared. With the
		// title left in the bar, the cursor came back to "New note:
		// <title>", and a second Enter made a second note.
		if land := os.Getenv("SWOOP_LAND"); land != "" {
			if err := os.WriteFile(land, []byte(filepath.Base(p)+"\n"), 0o600); err != nil {
				return err
			}
		}
		return settings.Edit(p)
	}
	p, err := path(id)
	if err != nil {
		return err
	}
	switch action {
	case "":
		return settings.Edit(p)
	case "open":
		return openFile(p)
	case "copy":
		text, err := readCapped(p)
		if err != nil {
			return err
		}
		if err := paste.Copy(text); err != nil {
			return err
		}
		paste.Tell("Copied " + paste.Short(Title(firstLine(text), id)))
		return nil
	case "reveal":
		return reveal(p)
	case "delete":
		_, err := remove(dir(), id)
		return err
	}
	return fmt.Errorf("no action %q for a note", action)
}
