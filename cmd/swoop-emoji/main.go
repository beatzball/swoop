// swoop-emoji is the emoji and symbols extension: one root row, Emoji &
// Symbols, that opens a view. Typing searches names and keywords; Enter
// pastes the character into the app in front (see internal/paste), in
// the skin tone chosen in Settings; ctrl-k copies it, or pastes it in
// one of the six tones for this once. What was used most recently leads
// the view, from the launcher's usage log.
//
//	swoop-emoji list                 the root row
//	swoop-emoji view emoji [query]   the characters that match
//	swoop-emoji preview <char>       its name, code points, keywords, tones
//	swoop-emoji actions <char>       Paste, Copy, and the six tones
//	swoop-emoji run <char> [action]  paste, copy, or tone0..tone5
//
// The id of a row is the character itself, without a tone: the tone is
// applied when it is used, so a change of the setting changes every row.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/usage"
)

// viewID is the root row's id, and the view it opens.
const viewID = "emoji"

// recentRows is how many of the most recently used characters lead the
// view.
const recentRows = 12

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
		// The hot path: one fixed row, and the table is never parsed.
		err = protocol.Write(os.Stdout, []protocol.Item{{ID: viewID, Kind: "view", Icon: "😀", Title: "Emoji & Symbols", Subtitle: "search by name, Enter pastes"}})
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
		fmt.Fprintln(os.Stderr, "swoop-emoji:", err)
		os.Exit(1)
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-emoji list | view emoji [query] | preview <char> | actions <char> | run <char> [paste|copy|tone0..tone5]")
	os.Exit(2)
}

// view prints the matches, the most recently used first, each in the
// default tone.
func view(query string) error {
	tone := defaultTone()
	matches := search(load(), custom(), query)
	items := make([]protocol.Item, len(matches))
	for i, e := range matches {
		items[i] = protocol.Item{ID: e.Char, Kind: "emoji", Icon: e.WithTone(tone), Title: e.Name, Subtitle: e.Group}
	}
	return protocol.Write(os.Stdout, usage.Front(items, recent(), recentRows))
}

// recent is the characters used most recently, newest first, from the
// usage log: swoop-run records every Enter under the row's routed id,
// ext/emoji/<char>.
func recent() []string {
	entries, err := usage.Load()
	if err != nil {
		return nil
	}
	const prefix = "ext/emoji/"
	var mine []usage.Entry
	for _, e := range entries {
		if c, ok := strings.CutPrefix(e.ID, prefix); ok && c != viewID {
			e.ID = c
			mine = append(mine, e)
		}
	}
	return usage.Recent(mine, recentRows)
}

func preview(id string) error {
	if id == viewID {
		fmt.Printf("Every emoji, flag and symbol, by name and keyword.\n\n"+
			"Enter pastes into the app you came from. ctrl-k: copy, or paste in a\nskin tone. The default tone is a setting.\n\n"+
			"Words of your own: %s, one line per\ncharacter, `😀 grin happy`.\n", tilde(keywordsPath()))
		return nil
	}
	e, ok := find(load(), id)
	if !ok {
		return fmt.Errorf("no emoji or symbol %q", id)
	}
	tone := defaultTone()
	fmt.Printf("\x1b[1m%s  %s\x1b[22m\n\n%s\n%s\n", e.WithTone(tone), e.Name, e.Group, codepoints(e.WithTone(tone)))
	kw := e.Keywords
	if extra := custom()[key(e.Char)]; len(extra) > 0 {
		kw = append(append([]string{}, kw...), extra...)
	}
	if len(kw) > 0 {
		fmt.Printf("\n%s\n", strings.Join(kw, " · "))
	}
	if e.Tone != "" {
		var all []string
		for n := range toneNames {
			all = append(all, e.WithTone(n))
		}
		fmt.Printf("\nSkin tones: %s\nctrl-k to paste one; the default is in Settings.\n", strings.Join(all, " "))
	}
	return nil
}

// codepoints is the character as U+ numbers, the way to find it in any
// other tool.
func codepoints(s string) string {
	var cps []string
	for _, r := range s {
		cps = append(cps, fmt.Sprintf("U+%04X", r))
	}
	return strings.Join(cps, " ")
}

// tilde shows the home directory as ~, so no home path reaches the
// screen.
func tilde(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// actions is the ctrl-k menu: Paste, Copy, and for a character that
// takes a skin tone, each of the six, the default marked. The root row
// has none.
func actions(id string) error {
	if id == viewID {
		return nil
	}
	e, ok := find(load(), id)
	if !ok {
		return nil
	}
	tone := defaultTone()
	items := []protocol.Item{
		{ID: "paste", Kind: "action", Icon: e.WithTone(tone), Title: "Paste", Subtitle: "Enter"},
		{ID: "copy", Kind: "action", Icon: e.WithTone(tone), Title: "Copy", Subtitle: "to the clipboard"},
	}
	if e.Tone != "" {
		for n, name := range toneNames {
			sub := name
			if n == tone {
				sub += "  default"
			}
			items = append(items, protocol.Item{ID: "tone" + strconv.Itoa(n), Kind: "action", Icon: e.WithTone(n), Title: "Paste " + e.WithTone(n), Subtitle: sub})
		}
	}
	return protocol.Write(os.Stdout, items)
}

func run(id, action string) error {
	e, ok := find(load(), id)
	if !ok {
		return fmt.Errorf("no emoji or symbol %q", id)
	}
	text := e.WithTone(defaultTone())
	switch {
	case action == "" || action == "open" || action == "paste":
	case action == "copy":
		return paste.Copy(text)
	case strings.HasPrefix(action, "tone"):
		n, err := strconv.Atoi(strings.TrimPrefix(action, "tone"))
		if err != nil || n < 0 || n >= len(toneNames) {
			return fmt.Errorf("no tone %q", action)
		}
		text = e.WithTone(n)
	default:
		return fmt.Errorf("no action %q for an emoji", action)
	}
	note, err := paste.Paste(text)
	if err != nil {
		return err
	}
	if note != "" {
		paste.Tell(note)
	}
	return nil
}
