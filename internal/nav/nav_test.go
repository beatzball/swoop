package nav

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEnterOnActionRowBecomesRun(t *testing.T) {
	st := &State{}
	got := Enter(st, "/Applications/Safari.app", "app", "Safari", "saf", 1, "swoop-run '/Applications/Safari.app'")
	if got != "become:swoop-run '/Applications/Safari.app'" {
		t.Fatalf("got %q", got)
	}
	if len(st.Stack) != 0 {
		t.Fatal("an action row must not push a pane")
	}
}

func TestEnterOnNothingIsIgnored(t *testing.T) {
	if got := Enter(&State{}, "", "", "", "zzz", 0, ""); got != "ignore" {
		t.Fatalf("got %q", got)
	}
}

func TestEnterOnViewRowPushesAndLoads(t *testing.T) {
	st := &State{}
	got := Enter(st, "ext/define/define", "view", "Define Word", "de", 3, "")
	want := "clear-query+disable-search+change-prompt(Define Word > )+reload-sync(swoop-nav rows {q})"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0] != (Frame{View: "ext/define/define", Title: "Define Word", Query: "de", Pos: 3}) {
		t.Fatalf("pushed frame wrong: %+v", st.Stack)
	}
}

func TestEscClearsThenPopsThenCloses(t *testing.T) {
	st := &State{Stack: []Frame{{View: "ext/define/define", Title: "Define Word", Query: "de", Pos: 3}}}
	if got := Esc(st, "wo"); got != "clear-query" {
		t.Fatalf("text in the bar: got %q", got)
	}
	got := Esc(st, "")
	want := "enable-search+change-prompt(  )+reload-sync(swoop-nav rows {q})+change-query(de)+wait+pos(3)"
	if got != want {
		t.Fatalf("pop: got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 0 {
		t.Fatal("pop must remove the frame")
	}
	if got := Esc(st, ""); got != "abort" {
		t.Fatalf("root with an empty bar: got %q", got)
	}
}

func TestChangeReloadsEverywhere(t *testing.T) {
	for _, st := range []*State{{}, {Stack: []Frame{{View: "x"}}}} {
		if got := Change(st); got != "reload-sync(swoop-nav rows {q})" {
			t.Fatalf("got %q", got)
		}
	}
}

func TestWrapPicksASafeDelimiter(t *testing.T) {
	cases := map[string]string{
		"Define Word":   "change-prompt(Define Word)",
		"a (b)":         "change-prompt[a (b)]",
		"a (b) [c]":     "change-prompt{a (b) [c]}",
		"a (b) [c] {d}": "change-prompt<a (b) [c] {d}>",
	}
	for arg, want := range cases {
		if got := wrap("change-prompt", arg); got != want {
			t.Errorf("wrap(%q) = %q, want %q", arg, got, want)
		}
	}
	all := "() [] {} <> ~~"
	if got := wrap("change-prompt", all); strings.Count(got, ")") != 1 {
		t.Errorf("when every pair is used, the closing paren must be stripped: %q", got)
	}
}

func TestShellQuote(t *testing.T) {
	if got := ShellQuote("it's"); got != `'it'\''s'` {
		t.Fatalf("got %s", got)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	st, err := Load(path)
	if err != nil || st.Top() != nil {
		t.Fatalf("missing file must be the root: %+v %v", st, err)
	}
	st.Stack = append(st.Stack, Frame{View: "v", Title: "T", Query: "q", Pos: 2})
	if err := Save(path, st); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path)
	if err != nil || len(back.Stack) != 1 || back.Stack[0] != st.Stack[0] {
		t.Fatalf("round trip lost state: %+v %v", back, err)
	}
}
