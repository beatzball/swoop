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
		return "become:cleanup; swoop-run " + ShellQuote(target)
	}
	return "become:cleanup; swoop-run " + ShellQuote(target) + " " + ShellQuote(action)
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
	want := "enable-search+change-prompt(  )+change-preview-window(right,45%,border-left,nowrap)+change-preview(swoop-preview {1})+change-query(de)+reload-sync(swoop-nav rows {q})+wait+pos(3)"
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
	want := "execute-silent(swoop-run 'ext/clipboard/17' 'delete')+disable-search+change-prompt(Clipboard History > )+change-preview-window(right,45%,border-left,nowrap)+change-preview(swoop-preview {1})+change-query(he)+reload-sync(swoop-nav rows {q})+wait+pos(2)"
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
		if got := Wrap("change-prompt", arg); got != want {
			t.Errorf("wrap(%q) = %q, want %q", arg, got, want)
		}
	}
	all := "() [] {} <> ~~"
	if got := Wrap("change-prompt", all); strings.Count(got, ")") != 1 {
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

func TestTabOpensTheAIPaneAndKeepsTheText(t *testing.T) {
	st := &State{}
	got := Ask(st, "why is the sky blue", 3)
	want := "disable-search+change-prompt(Ask AI > )+change-preview-window(right,45%,border-left,wrap,follow)+reload-sync(swoop-nav rows)+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0].Kind != "ai" || st.Stack[0].View != AIView || st.Stack[0].Query != "why is the sky blue" || st.Stack[0].Pos != 3 {
		t.Fatalf("stack: %+v", st.Stack)
	}
	if got := Ask(st, "", 1); got != "ignore" {
		t.Fatalf("Tab inside the pane does nothing: got %q", got)
	}
	if got := Change(st); got != "ignore" {
		t.Fatalf("typing in the pane must not reload: got %q", got)
	}
}

func TestTabOnAnEmptyBarOpensThePaneToo(t *testing.T) {
	st := &State{}
	if got := Ask(st, "", 1); got == "ignore" {
		t.Fatal("an empty bar still opens the pane")
	}
	if len(st.Stack) != 1 || st.Stack[0].Query != "" {
		t.Fatalf("stack: %+v", st.Stack)
	}
}

func TestEnterInTheAIPaneSendsThenClears(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "ai", View: AIView, Title: AITitle}}}
	if _, ok := AISendTarget(st, "ext/ai/new", "  "); ok {
		t.Fatal("nothing to send")
	}
	if _, ok := AISendTarget(st, "", "why"); ok {
		t.Fatal("no row, nothing to send to")
	}
	if target, ok := AISendTarget(st, "ext/ai/new", "why"); !ok || target != "new" {
		t.Fatalf("New row sends to new: %q %v", target, ok)
	}
	if target, ok := AISendTarget(st, "ext/ai/20260925-1", "and then"); !ok || target != "20260925-1" {
		t.Fatalf("a conversation row sends to itself: %q %v", target, ok)
	}
	if _, ok := AISendTarget(&State{}, "ext/ai/new", "why"); ok {
		t.Fatal("only inside the pane")
	}
	if got := AfterSend(); got != "clear-query+reload-sync(swoop-nav rows)+wait+pos(2)+refresh-preview" {
		t.Fatalf("got %q", got)
	}
	if got := Enter(st, "ext/ai/20260925-1", "conversation", "earlier", "and then", 2, run); got != "ignore" {
		t.Fatalf("Enter itself neither runs nor pushes in the pane: %q", got)
	}
	if len(st.Stack) != 1 {
		t.Fatalf("Enter must not push: %+v", st.Stack)
	}
}

func TestEscFromTheAIPaneRestoresTheBar(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "ai", View: AIView, Title: AITitle, Query: "why is the sky blue", Pos: 2}}}
	if got := Esc(st, "draft"); got != "clear-query" {
		t.Fatalf("text first: %q", got)
	}
	got := Esc(st, "")
	for _, part := range []string{"enable-search", "change-preview-window(right,45%,border-left,nowrap)", "change-query(why is the sky blue)", "pos(2)"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
	if len(st.Stack) != 0 {
		t.Fatalf("should be at the root: %+v", st.Stack)
	}
}

func TestPoppingActionsInsideTheAIPaneKeepsItsWindow(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "ai", View: AIView, Title: AITitle, Query: "", Pos: 1},
		{Kind: "actions", View: "ext/ai/20260925-1", Title: "earlier", Query: "draft", Pos: 2},
	}}
	got := Esc(st, "")
	if !strings.Contains(got, "change-preview-window(right,45%,border-left,wrap,follow)") || !strings.Contains(got, "change-prompt(Ask AI > )") {
		t.Fatalf("got %q", got)
	}
}
