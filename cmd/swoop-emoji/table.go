// The table and the search. table.tsv is every emoji from Unicode's
// emoji-test.txt with its CLDR name and keywords, and a few hundred
// symbols; gen.go writes it. It is embedded, so it costs nothing until a
// view reads it: `list` at the root never does.
package main

import (
	_ "embed"
	"github.com/beatzball/swoop/internal/match"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/beatzball/swoop/internal/settings"
)

//go:embed table.tsv
var table string

// Entry is one character.
type Entry struct {
	Char     string   // the character, fully qualified; also the row's id
	Name     string   // CLDR's name, or Unicode's for a symbol without one
	Group    string   // Smileys & Emotion, Arrows, ...
	Tone     string   // the character with ~ where a skin tone goes; "" when it takes none
	Keywords []string // CLDR's keywords, lower case
}

// load parses the table.
func load() []Entry {
	entries := make([]Entry, 0, 2600)
	for _, line := range strings.Split(table, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 5 {
			continue
		}
		e := Entry{Char: f[0], Name: f[1], Group: f[2], Tone: f[3]}
		if f[4] != "" {
			e.Keywords = strings.Split(f[4], "|")
		}
		entries = append(entries, e)
	}
	return entries
}

// toneNames are the six choices, 0 to 5.
var toneNames = []string{"no tone", "light", "medium-light", "medium", "medium-dark", "dark"}

// WithTone is the character in skin tone n, 1 to 5 from light to dark.
// 0, a tone out of range, or a character that takes none is the
// character as it is.
func (e Entry) WithTone(n int) string {
	if e.Tone == "" || n < 1 || n > 5 {
		return e.Char
	}
	return strings.ReplaceAll(e.Tone, "~", string(rune(0x1F3FA+n)))
}

// defaultTone is the tone from Settings, 0 when unset or not 0 to 5.
func defaultTone() int {
	n, err := strconv.Atoi(settings.Get(settings.Skin, settings.SkinDefault))
	if err != nil || n < 0 || n > 5 {
		return 0
	}
	return n
}

// key is how a character is looked up: without the emoji presentation
// selector, which a hand-written keywords file may or may not have.
func key(s string) string { return strings.ReplaceAll(s, "️", "") }

// find returns the entry for a character.
func find(entries []Entry, char string) (Entry, bool) {
	k := key(char)
	for _, e := range entries {
		if key(e.Char) == k {
			return e, true
		}
	}
	return Entry{}, false
}

// keywordsPath is the user's own keywords, ~/.config/swoop/emoji.keywords.
func keywordsPath() string {
	dir := settings.Dir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "emoji.keywords")
}

// custom reads the user's keywords: one line per character, the
// character then its words, `😀 grin happy`. A line starting with # is a
// comment. No file is no keywords.
func custom() map[string][]string {
	data, err := os.ReadFile(keywordsPath())
	if err != nil {
		return nil
	}
	return parseCustom(string(data))
}

func parseCustom(text string) map[string][]string {
	out := map[string][]string{}
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(strings.ToLower(line))
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		out[key(f[0])] = append(out[key(f[0])], f[1:]...)
	}
	return out
}

// words splits text into lower-case words at anything not a letter or a
// digit, so "flag: Japan" is flag and japan and "right-pointing" is right
// and pointing. "+1" keeps its sign: a keyword people type as is.
func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '+'
	})
}

// Match ranks, best first: the name exactly (or the character itself);
// the name's start, or every word typed starting a word of the name; and
// every word typed starting a word of the name, the keywords, or the
// user's own keywords. The name's start is not ranked above its other
// words: "heart" must find the red heart, whose name ends with it, ahead
// of "heart hands".
const (
	exact = iota
	inName
	anyWords
	noMatch
)

func score(e Entry, extra []string, query string, tokens []string) int {
	name := strings.ToLower(e.Name)
	switch {
	case query == name || key(query) == key(e.Char):
		return exact
	case strings.HasPrefix(name, query):
		return inName
	}
	nw := words(e.Name)
	if allPrefix(tokens, nw) {
		return inName
	}
	all := nw
	for _, k := range e.Keywords {
		all = append(all, words(k)...)
	}
	all = append(all, extra...)
	if allPrefix(tokens, all) {
		return anyWords
	}
	return noMatch
}

// allPrefix says whether every token starts one of the words.
func allPrefix(tokens, ws []string) bool {
	for _, t := range tokens {
		found := false
		for _, w := range ws {
			if strings.HasPrefix(w, t) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// search returns the entries that match query, best first. Within the
// name ranks a shorter name leads, so "heart" puts the red heart's plain
// name ahead of "heart with arrow"; among keyword matches the table's own
// order holds, which is the order of a keyboard's picker. After those
// come the entries only the matcher every view shares finds
// (internal/match): a word inside a word of the name or a keyword, or
// its letters in order in the name, "rckt" for rocket, the best of them
// first. The keywords count as typed only: there are thousands of them,
// and the letters of a short word are spread in some of them. An empty
// query is the whole table in that order.
func search(entries []Entry, extra map[string][]string, query string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return entries
	}
	tokens := words(query)
	if len(tokens) == 0 {
		// Only punctuation typed: a character looked up as itself.
		tokens = []string{query}
	}
	type hit struct {
		e     Entry
		score int
	}
	var hits []hit
	q := match.New(query)
	var loose match.Best[Entry]
	for _, e := range entries {
		mine := extra[key(e.Char)]
		if s := score(e, mine, query, tokens); s != noMatch {
			hits = append(hits, hit{e, s})
		} else if s, ok := q.ScoreProse(strings.Join(e.Keywords, " ")+" "+strings.Join(mine, " "), e.Name); ok {
			loose.Add(e, s)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.score != b.score {
			return a.score < b.score
		}
		if a.score < anyWords {
			return len(a.e.Name) < len(b.e.Name)
		}
		return false
	})
	out := make([]Entry, len(hits))
	for i, h := range hits {
		out[i] = h.e
	}
	return append(out, loose.Rows()...)
}
