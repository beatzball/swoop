package match

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
)

// A word matches when its letters are in the text in that order, in any
// case.
func TestSubsequence(t *testing.T) {
	for _, c := range []struct {
		query string
		texts []string
		want  bool
	}{
		{"", []string{"API key"}, true},
		{"   ", []string{"API key"}, true},
		{"akey", []string{"API key"}, true},
		{"AKEY", []string{"api key"}, true},
		{"apikey", []string{"API key"}, true},
		{"nvm", []string{"Editor", "nvim"}, true},
		// The letters are there, in the wrong order.
		{"yeka", []string{"API key"}, false},
		{"keya", []string{"API key"}, false},
		// A letter typed twice must be there twice.
		{"keyy", []string{"API key"}, false},
		// A word sits inside one text, not across two.
		{"ornv", []string{"Editor", "nvim"}, false},
		// Letters outside ASCII are letters too.
		{"cafe", []string{"Café"}, false},
		{"café", []string{"CAFÉ au lait"}, true},
		{"cfé", []string{"Café"}, true},
	} {
		if _, got := New(c.query).Score(c.texts...); got != c.want {
			t.Errorf("Score(%q, %q) = %v, want %v", c.query, c.texts, got, c.want)
		}
	}
}

// Every word must match one of the texts, in any order (#196).
func TestWordOrder(t *testing.T) {
	for _, c := range []struct {
		query string
		texts []string
		want  bool
	}{
		{"api key", []string{"API key", "not set"}, true},
		{"key api", []string{"API key", "not set"}, true},
		{"KEY  Api", []string{"API key", "not set"}, true},
		{"ai key", []string{"API key", "not set", "ai"}, true},
		{"nvim", []string{"Editor", "nvim"}, true},
		{"edit nv", []string{"Editor", "nvim"}, true},
		{"editor vim emacs", []string{"Editor", "nvim"}, false},
	} {
		if _, got := New(c.query).Score(c.texts...); got != c.want {
			t.Errorf("Score(%q, %q) = %v, want %v", c.query, c.texts, got, c.want)
		}
	}
}

// The ranking rules, each one as a pair: the first row must beat the
// second.
func TestRanking(t *testing.T) {
	for _, c := range []struct {
		rule          string
		query         string
		better, worse []string
	}{
		{"exact beats spread out", "key", []string{"monkey"}, []string{"kelly"}},
		{"exact inside a word beats spread out from a word's start", "key", []string{"monkey"}, []string{"k-e-y"}},
		{"a word's start beats inside a word", "key", []string{"API key"}, []string{"monkey"}},
		{"a word's start beats inside a word, spread out", "akey", []string{"API key"}, []string{"zapikey"}},
		{"the title beats the subtitle", "nvim", []string{"nvim", "x"}, []string{"x", "nvim"}},
		{"a word's start in the subtitle beats inside the title", "key", []string{"x", "a key"}, []string{"monkey", "x"}},
		{"a shorter span beats a longer one", "aky", []string{"a key"}, []string{"ab key"}},
		{"two words: each rule counts for each word", "api key", []string{"API key"}, []string{"API monkey"}},
	} {
		q := New(c.query)
		b, ok1 := q.Score(c.better...)
		w, ok2 := q.Score(c.worse...)
		if !ok1 || !ok2 {
			t.Errorf("%s: %q must match both %q (%v) and %q (%v)", c.rule, c.query, c.better, ok1, c.worse, ok2)
			continue
		}
		if b <= w {
			t.Errorf("%s: %q scores %d in %q, %d in %q", c.rule, c.query, b, c.better, w, c.worse)
		}
	}
}

// A spread-out match must not cover the whole of a long text, and a very
// short word matches only as typed.
func TestNoiseGuard(t *testing.T) {
	for _, c := range []struct {
		query, text string
		want        bool
	}{
		{"akey", "API key", true},
		// The same letters, in order, over too many others.
		{"akey", "a big monkey", false},
		{"ctrl", "quick terminal", false},
		{"akey", "all the kings men eat yams", false},
		// Two letters with something in between are in every text.
		{"ai", "API", false},
		{"ai", "AI model", true},
		{"ky", "key", false},
	} {
		if _, got := New(c.query).Score(c.text); got != c.want {
			t.Errorf("Score(%q, %q) = %v, want %v", c.query, c.text, got, c.want)
		}
	}
}

// Prose matches only as typed, whatever its length, and ranks below the
// title.
func TestProse(t *testing.T) {
	q := New("akey")
	if _, ok := q.ScoreProse("the API key is here", "Title"); ok {
		t.Error("a spread-out match in prose")
	}
	q = New("bread")
	inTitle, ok1 := q.ScoreProse("flour, water", "Bread recipe")
	inProse, ok2 := q.ScoreProse("pack the bread knife", "Trip")
	if !ok1 || !ok2 || inTitle <= inProse {
		t.Errorf("title %d %v, prose %d %v", inTitle, ok1, inProse, ok2)
	}
}

func TestWord(t *testing.T) {
	if _, ok := Word("groc", "Groceries"); !ok {
		t.Error("groc is in Groceries")
	}
	if _, ok := Word("milk", "Groceries"); ok {
		t.Error("milk is not in Groceries")
	}
}

// Rows with the same score keep the order the view gave them, and the
// best come first.
func TestRank(t *testing.T) {
	q := New("key")
	rows := []string{"monkey", "turkey", "Key one", "donkey", "Key two", "nothing"}
	got := Rank(rows, func(s string) (int, bool) { return q.Score(s) })
	if want := []string{"Key one", "Key two", "monkey", "turkey", "donkey"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// No query: every row, as given.
	q = New("")
	if got := Rank(rows, func(s string) (int, bool) { return q.Score(s) }); !reflect.DeepEqual(got, rows) {
		t.Errorf("an empty query changed the rows: %v", got)
	}
}

// A pinned row keeps its place, whatever the others score.
func TestPin(t *testing.T) {
	var b Best[string]
	b.Pin("first")
	b.Add("low", 1)
	b.Add("high", 9)
	b.Pin("third")
	b.Add("mid", 5)
	b.Pin("last")
	if got, want := b.Rows(), []string{"first", "high", "mid", "third", "low", "last"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	var only Best[string]
	only.Pin("typed")
	if got := only.Rows(); !reflect.DeepEqual(got, []string{"typed"}) {
		t.Errorf("a pinned row alone: %v", got)
	}
}

// Items ranks protocol rows under their headers, and leaves out a header
// with no matching row.
func TestItems(t *testing.T) {
	items := []protocol.Item{
		{ID: "0", Kind: "toggle", Title: "Before any header: monkey"},
		{ID: "g1", Kind: GroupKind, Title: "Today"},
		{ID: "1", Kind: "toggle", Title: "Feed the monkey"},
		{ID: "2", Kind: "toggle", Title: "Key for the shed"},
		{ID: "g2", Kind: GroupKind, Title: "Tomorrow key"},
		{ID: "3", Kind: "toggle", Title: "Nothing"},
		{ID: "g3", Kind: GroupKind, Title: "Later"},
		{ID: "4", Kind: "toggle", Title: "Nothing", Subtitle: "a key in the subtitle"},
	}
	ids := func(items []protocol.Item) []string {
		var out []string
		for _, it := range items {
			out = append(out, it.ID)
		}
		return out
	}
	if got, want := ids(Items(New("key"), items)), []string{"0", "g1", "2", "1", "g3", "4"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := Items(New(" "), items); !reflect.DeepEqual(got, items) {
		t.Errorf("an empty query changed the rows: %v", ids(got))
	}
}

// rows is a list the size of a big view: a few thousand rows with a
// title and a subtitle.
func rows(n int) []protocol.Item {
	names := []string{"API key", "Editor", "Launch at login", "Clipboard history", "Feed the monkey", "Window: left half", "Snippet signature", "Notes folder"}
	items := make([]protocol.Item, n)
	for i := range items {
		items[i] = protocol.Item{ID: fmt.Sprint(i), Kind: "text", Title: fmt.Sprintf("%s %d", names[i%len(names)], i), Subtitle: "the value of row " + names[(i+3)%len(names)]}
	}
	return items
}

// A view answers on every keystroke: a few thousand rows must filter in
// well under a frame (16 ms). The limit here is loose, for a slow runner;
// BenchmarkItems has the number.
func TestFastEnough(t *testing.T) {
	items := rows(5000)
	q := New("akey row")
	start := time.Now()
	const rounds = 10
	for range rounds {
		Items(q, items)
	}
	if per := time.Since(start) / rounds; per > 16*time.Millisecond {
		t.Errorf("5000 rows took %v, more than a frame", per)
	}
}

func BenchmarkItems(b *testing.B) {
	items := rows(5000)
	for _, query := range []string{"key", "akey", "akey row", "zzzz"} {
		q := New(query)
		b.Run(query, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Items(q, items)
			}
		})
	}
}
