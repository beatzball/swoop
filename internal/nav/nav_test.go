package nav

import (
	"path/filepath"
	"strings"
	"testing"
)

// run is the test's stand-in for the shell command that performs a row.
func run(target, action string) string {
	// The exit command: it does more than run, which is why a refresh
	// action must not use it. The tests check the extra never leaks in.
	if action == "" {
		return "cleanup; swoop-run " + ShellQuote(target)
	}
	return "cleanup; swoop-run " + ShellQuote(target) + " " + ShellQuote(action)
}

func TestEnterOnActionRowBecomesRun(t *testing.T) {
	st := &State{}
	got := Enter(st, "/Applications/Safari.app", "app", "Safari", "saf", 1, run)
	if got != "become:cleanup; swoop-run '/Applications/Safari.app'" {
		t.Fatalf("got %q", got)
	}
	if len(st.Stack) != 0 {
		t.Fatal("an action row must not push a pane")
	}
}

func TestEnterOnNothingIsIgnored(t *testing.T) {
	if got := Enter(&State{}, "", "", "", "zzz", 0, run); got != "ignore" {
		t.Fatalf("got %q", got)
	}
}

func TestEnterOnViewRowPushesAndLoads(t *testing.T) {
	st := &State{}
	got := Enter(st, "ext/define/define", "view", "Define Word", "de", 3, run)
	want := "clear-query+disable-search+change-prompt(Define Word > )+reload-sync(swoop-nav rows {q})+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0] != (Frame{Kind: "view", View: "ext/define/define", Title: "Define Word", Query: "de", Pos: 3}) {
		t.Fatalf("pushed frame wrong: %+v", st.Stack)
	}
}

func TestEscClearsThenPopsThenCloses(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/define/define", Title: "Define Word", Query: "de", Pos: 3}}}
	if got := Esc(st, "wo"); got != "clear-query" {
		t.Fatalf("text in the bar: got %q", got)
	}
	got := Esc(st, "")
	want := "enable-search+change-prompt(  )+change-preview(swoop-preview {1})+change-query(de)+reload-sync(swoop-nav rows {q})+wait+pos(3)"
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

func TestActionsPushesAPaneAndKeepsThePreviewOnTheTarget(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/clipboard/clipboard", Title: "Clipboard History", Query: "", Pos: 1}}}
	got := Actions(st, "ext/clipboard/17", "text", "hello", "he", 2)
	want := "clear-query+disable-search+change-prompt(hello actions > )+reload-sync(swoop-nav rows {q})+first+change-preview(swoop-preview 'ext/clipboard/17')"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 2 || st.Stack[1].Kind != "actions" || st.Stack[1].View != "ext/clipboard/17" {
		t.Fatalf("frame wrong: %+v", st.Stack)
	}
	if got := Actions(&State{}, "", "", "", "", 0); got != "ignore" {
		t.Fatalf("no row: got %q", got)
	}
}

func TestEnterOnRefreshActionRunsAndReturnsToTheViewBelow(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "view", View: "ext/clipboard/clipboard", Title: "Clipboard History", Query: "", Pos: 1},
		{Kind: "actions", View: "ext/clipboard/17", Title: "hello", Query: "he", Pos: 2},
	}}
	got := Enter(st, "delete", "refresh", "Delete", "", 2, run)
	want := "execute-silent(swoop-run 'ext/clipboard/17' 'delete')+disable-search+change-prompt(Clipboard History > )+change-preview(swoop-preview {1})+change-query(he)+reload-sync(swoop-nav rows {q})+wait+pos(2)"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0].Kind != "view" {
		t.Fatalf("should be back in the view: %+v", st.Stack)
	}
}

func TestEnterOnExitingActionBecomesRunWithTheAction(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "actions", View: "/Applications/Safari.app", Title: "Safari", Query: "saf", Pos: 1}}}
	got := Enter(st, "reveal", "action", "Reveal in Finder", "", 2, run)
	if got != "become:cleanup; swoop-run '/Applications/Safari.app' 'reveal'" {
		t.Fatalf("got %q", got)
	}
}

func TestEscFromActionsReturnsToTheViewBelow(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "view", View: "ext/clipboard/clipboard", Title: "Clipboard History", Query: "", Pos: 1},
		{Kind: "actions", View: "ext/clipboard/17", Title: "hello", Query: "he", Pos: 2},
	}}
	got := Esc(st, "")
	if !strings.HasPrefix(got, "disable-search+change-prompt(Clipboard History > )") || !strings.HasSuffix(got, "change-query(he)+reload-sync(swoop-nav rows {q})+wait+pos(2)") {
		t.Fatalf("got %q", got)
	}
}

func TestChangeReloadsEverywhere(t *testing.T) {
	for _, st := range []*State{{}, {Stack: []Frame{{Kind: "view", View: "x"}}}} {
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
	st.Stack = append(st.Stack, Frame{Kind: "view", View: "v", Title: "T", Query: "q", Pos: 2})
	if err := Save(path, st); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path)
	if err != nil || len(back.Stack) != 1 || back.Stack[0] != st.Stack[0] {
		t.Fatalf("round trip lost state: %+v %v", back, err)
	}
}
