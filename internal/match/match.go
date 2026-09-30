// Package match is the one matcher every view filters its own rows with.
// At the root fzf matches; inside a view its matching is off, and the view
// keeps the rows that answer to the text in the bar (#201).
//
// The text is split on spaces. A row matches when every word matches one
// of its texts (title, subtitle, hidden words), in any order and any case.
// A word matches a text when its letters are in the text in that order:
// "akey" matches "API key". A row gets a score, so the best rows come
// first, and rows with the same score keep the order the view gave them.
//
// The ranking rules are the constants below and nothing else. Tune them
// there.
package match

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/beatzball/swoop/internal/protocol"
)

// The ranking rules. A word's score in a text is the sum of what applies;
// a row's score is the sum over its words, each word in the text where it
// scores best. The sizes are picked so that a rule higher in this list
// beats all the rules below it together.
const (
	// Exact: the word's letters sit side by side in the text, as typed.
	// An exact match beats one spread out over the text.
	Exact = 400
	// WordStart: the match begins where a word of the text begins. "key"
	// in "API key" beats "key" in "monkey".
	WordStart = 200
	// Title: the match is in the row's first text, its title. A title
	// match beats a match in the subtitle or the hidden words.
	Title = 100
	// Gap is taken off for every letter of the text that a spread-out
	// match steps over, MaxGap at the most: a shorter span beats a longer
	// one.
	Gap    = 1
	MaxGap = 99
)

// The guard against noise. Any few letters are somewhere in a long text
// in the right order, so a spread-out match has limits.
const (
	// Spread: the span of a spread-out match, from its first letter to
	// its last, is at most Spread times the word's length. "akey" may
	// cover 8 letters of the text, so it finds "API key" (7 letters from
	// A to y), and does not find "a big monkey" (12). At 3 times, "ctrl"
	// found "quick terminal", which is the noise this is here to stop.
	Spread = 2
	// MinSpread: a word shorter than this matches only as typed. Two
	// letters with anything in between are in almost every text.
	MinSpread = 3
)

// GroupKind is the kind of a header row, a group's name over its rows
// (see internal/nav).
const GroupKind = "group"

// Query is the text in the bar, split into its words.
type Query struct {
	words []string // lower case
}

// New splits the text in the bar on spaces.
func New(query string) Query {
	return Query{words: strings.Fields(strings.ToLower(query))}
}

// Empty says whether there is no word to match: every row matches.
func (q Query) Empty() bool { return len(q.words) == 0 }

// Words are the query's words, in lower case.
func (q Query) Words() []string { return q.words }

// Score says whether a row with these texts matches, and how well: more
// is better. The first text is the title. Every word must match one
// text; a word cannot run from the end of one text into the next. An
// empty query matches every row, with the same score.
func (q Query) Score(texts ...string) (score int, ok bool) {
	return q.score("", texts)
}

// ScoreProse is Score for a row that also has a long text: a note's body,
// a clipboard entry. A word matches prose only as typed, side by side;
// the letters of any word are spread somewhere in a page of text.
func (q Query) ScoreProse(prose string, texts ...string) (score int, ok bool) {
	return q.score(prose, texts)
}

func (q Query) score(prose string, texts []string) (int, bool) {
	if len(q.words) == 0 {
		return 0, true
	}
	// Most rows have two or three texts; no allocation for those.
	var buf [4]string
	lower := buf[:0]
	for _, t := range texts {
		lower = append(lower, strings.ToLower(t))
	}
	prose = strings.ToLower(prose)
	total := 0
	for _, w := range q.words {
		best, found := 0, false
		for i, t := range lower {
			s, ok := word(w, t, true)
			if !ok {
				continue
			}
			if i == 0 {
				s += Title
			}
			if !found || s > best {
				best, found = s, true
			}
		}
		if prose != "" {
			if s, ok := word(w, prose, false); ok && (!found || s > best) {
				best, found = s, true
			}
		}
		if !found {
			return 0, false
		}
		total += best
	}
	return total, true
}

// Word says whether one word matches one text, and how well, by the rules
// above without Title. It is for a view that must know which text a word
// was found in.
func Word(w, text string) (score int, ok bool) {
	return word(strings.ToLower(w), strings.ToLower(text), true)
}

// word scores w in t, both lower case. With spread off, only a match as
// typed counts.
func word(w, t string, spread bool) (int, bool) {
	if w == "" {
		return 0, true
	}
	if at := strings.Index(t, w); at >= 0 {
		// The best place the word sits as typed: one at a word's start,
		// if there is one.
		for {
			if wordStart(t, at) {
				return Exact + WordStart, true
			}
			next := strings.Index(t[at+1:], w)
			if next < 0 {
				return Exact, true
			}
			at += 1 + next
		}
	}
	if !spread {
		return 0, false
	}
	wr := []rune(w)
	if len(wr) < MinSpread {
		return 0, false
	}
	tr := []rune(t)
	best, found := 0, false
	// From every place the first letter is, the soonest place the rest
	// follow, no further than the guard allows. The best of those wins.
	for i := 0; i+len(wr) <= len(tr); i++ {
		if tr[i] != wr[0] {
			continue
		}
		end := min(i+Spread*len(wr), len(tr))
		j, k := i+1, 1
		for ; j < end && k < len(wr); j++ {
			if tr[j] == wr[k] {
				k++
			}
		}
		if k < len(wr) {
			continue
		}
		s := -min(Gap*(j-i-len(wr)), MaxGap)
		if i == 0 || !wordRune(tr[i-1]) {
			s += WordStart
		}
		if !found || s > best {
			best, found = s, true
		}
	}
	return best, found
}

// wordStart says whether byte at of t begins a word: the text starts
// there, or what is before it is not a letter or a digit.
func wordStart(t string, at int) bool {
	if at == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(t[:at])
	return !wordRune(r)
}

func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// Best collects a view's rows and gives them back best first. A view adds
// each row that matched with its score, and pins the rows it made from
// the text typed (Add task, Use what you typed), which always show and
// stay where they are.
type Best[T any] struct {
	rows   []T
	scores []int
	pins   []pin[T]
}

type pin[T any] struct {
	row T
	at  int // its place among every row given, pinned or not
}

// Add takes a row that matched, with its score.
func (b *Best[T]) Add(row T, score int) {
	b.rows = append(b.rows, row)
	b.scores = append(b.scores, score)
}

// Pin takes a row that shows whatever the score of the others: it keeps
// the place it has now, counted from the top.
func (b *Best[T]) Pin(row T) {
	b.pins = append(b.pins, pin[T]{row, len(b.rows) + len(b.pins)})
}

// Rows are the rows, the best score first; rows with the same score keep
// the order they were added in, and a pinned row keeps its place.
func (b *Best[T]) Rows() []T {
	if len(b.rows)+len(b.pins) == 0 {
		return nil
	}
	order := make([]int, len(b.rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return b.scores[order[i]] > b.scores[order[j]] })
	out := make([]T, 0, len(b.rows)+len(b.pins))
	next := 0
	for _, i := range order {
		for next < len(b.pins) && b.pins[next].at == len(out) {
			out = append(out, b.pins[next].row)
			next++
		}
		out = append(out, b.rows[i])
	}
	for ; next < len(b.pins); next++ {
		out = append(out, b.pins[next].row)
	}
	return out
}

// Rank keeps the rows score says match, best first.
func Rank[T any](rows []T, score func(T) (int, bool)) []T {
	var b Best[T]
	for _, r := range rows {
		if s, ok := score(r); ok {
			b.Add(r, s)
		}
	}
	return b.Rows()
}

// Items keeps the protocol rows that match on their title or subtitle,
// best first. A row of kind "group" is a header over the rows below it:
// the rows are ranked under their own header, and a header with no
// matching row is left out. An empty query gives the rows back as they
// are.
func Items(q Query, items []protocol.Item) []protocol.Item {
	if q.Empty() {
		return items
	}
	out := make([]protocol.Item, 0, len(items))
	var header *protocol.Item
	var b Best[protocol.Item]
	flush := func() {
		rows := b.Rows()
		if len(rows) > 0 && header != nil {
			out = append(out, *header)
		}
		out = append(out, rows...)
		b = Best[protocol.Item]{}
	}
	for i, it := range items {
		if it.Kind == GroupKind {
			flush()
			header = &items[i]
			continue
		}
		if s, ok := q.Score(it.Title, it.Subtitle); ok {
			b.Add(it, s)
		}
	}
	flush()
	return out
}
