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
	want := "enable-search+change-prompt(  )+change-preview-window(right,58%,border-left,nowrap)+change-preview(swoop-preview {1})+change-query(de)+reload-sync(swoop-nav rows {q})+rebind(result-final)"
	if got != want {
		t.Fatalf("pop: got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 0 {
		t.Fatal("pop must remove the frame")
	}
	if st.Land == nil || *st.Land != (Landing{Query: "de", Pos: 3}) {
		t.Fatalf("the cursor goes back to row 3 once the root lands: %+v", st.Land)
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
	want := "execute-silent(swoop-run 'ext/clipboard/17' 'delete')+disable-search+change-prompt(Clipboard History > )+change-preview-window(right,58%,border-left,nowrap)+change-preview(swoop-preview {1})+change-query(he)+reload-sync(swoop-nav rows {q})+rebind(result-final)"
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
	if !strings.HasPrefix(got, "disable-search+change-prompt(Clipboard History > )") || !strings.HasSuffix(got, "change-query(he)+reload-sync(swoop-nav rows {q})+rebind(result-final)") {
		t.Fatalf("got %q", got)
	}
}

func TestChangeReloadsEverywhere(t *testing.T) {
	// At the root fzf's matching goes back on with every keystroke, so
	// deleting a keyword's space undoes the scope.
	if got := Change(&State{}, nil); got != "enable-search+reload-sync(swoop-nav rows {q})" {
		t.Fatalf("root: got %q", got)
	}
	st := &State{Stack: []Frame{{Kind: "view", View: "x"}}}
	if got := Change(st, nil); got != "reload-sync(swoop-nav rows {q})+first" {
		t.Fatalf("view: got %q", got)
	}
	// Inside a pane a keyword means nothing: cmd/swoop-nav never finds
	// one there, and a stray one is ignored.
	if got := Change(st, &Keyed{Rest: "ap", View: "ext/define/define", Title: "Define Word"}); got != "reload-sync(swoop-nav rows {q})+first" || len(st.Stack) != 1 {
		t.Fatalf("view with a keyword: got %q, stack %+v", got, st.Stack)
	}
}

func TestKeywordWithAViewOpensItWithTheRest(t *testing.T) {
	st := &State{}
	got := Change(st, &Keyed{Rest: "ap", View: "ext/define/define", Title: "Define Word"})
	want := "change-query(ap)+disable-search+change-prompt(Define Word > )+reload-sync(swoop-nav rows {q})+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	// The frame keeps an empty bar, so Esc comes back to the whole root.
	if len(st.Stack) != 1 || st.Stack[0] != (Frame{Kind: "view", View: "ext/define/define", Title: "Define Word", Pos: 1}) {
		t.Fatalf("stack: %+v", st.Stack)
	}
	if got := Esc(st, ""); !strings.Contains(got, "enable-search") || !strings.Contains(got, "change-query()") {
		t.Fatalf("Esc: got %q", got)
	}
}

func TestKeywordWithoutAViewScopesTheRoot(t *testing.T) {
	st := &State{}
	if got := Change(st, &Keyed{Rest: "left"}); got != "disable-search+reload-sync(swoop-nav rows {q})" {
		t.Fatalf("got %q", got)
	}
	if len(st.Stack) != 0 {
		t.Fatalf("nothing is pushed: %+v", st.Stack)
	}
}

func TestAIKeywordOpensTheAskPaneWithTheRest(t *testing.T) {
	st := &State{}
	got := Change(st, &Keyed{Rest: "why is the sky blue", View: AIView, Title: AITitle})
	want := "change-query(why is the sky blue)+disable-search+change-prompt(Ask AI > )+change-preview-window(right,58%,border-left,wrap,follow)+reload-sync(swoop-nav rows)+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0].Kind != "ai" || st.Stack[0].Query != "" {
		t.Fatalf("stack: %+v", st.Stack)
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
	want := "disable-search+change-prompt(Ask AI > )+change-preview-window(right,58%,border-left,wrap,follow)+reload-sync(swoop-nav rows)+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0].Kind != "ai" || st.Stack[0].View != AIView || st.Stack[0].Query != "why is the sky blue" || st.Stack[0].Pos != 3 {
		t.Fatalf("stack: %+v", st.Stack)
	}
	if got := Ask(st, "", 1); got != "ignore" {
		t.Fatalf("Tab inside the pane does nothing: got %q", got)
	}
	if got := Change(st, nil); got != "ignore" {
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
	if got := AfterSend(st); got != "clear-query+reload-sync(swoop-nav rows)+rebind(result-final)" {
		t.Fatalf("got %q", got)
	}
	if st.Land == nil || *st.Land != (Landing{Pos: 2, Refresh: true}) {
		t.Fatalf("the cursor goes to the conversation once the list lands: %+v", st.Land)
	}
	st.Land = nil
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
	for _, part := range []string{"enable-search", "change-preview-window(right,58%,border-left,nowrap)", "change-query(why is the sky blue)", "rebind(result-final)"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
	if len(st.Stack) != 0 {
		t.Fatalf("should be at the root: %+v", st.Stack)
	}
	if st.Land == nil || st.Land.Pos != 2 {
		t.Fatalf("back on row 2 once the root lands: %+v", st.Land)
	}
}

func TestPoppingActionsInsideTheAIPaneKeepsItsWindow(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "ai", View: AIView, Title: AITitle, Query: "", Pos: 1},
		{Kind: "actions", View: "ext/ai/20260925-1", Title: "earlier", Query: "draft", Pos: 2},
	}}
	got := Esc(st, "")
	if !strings.Contains(got, "change-preview-window(right,58%,border-left,wrap,follow)") || !strings.Contains(got, "change-prompt(Ask AI > )") {
		t.Fatalf("got %q", got)
	}
}

func TestDividerKeepsThePanesShape(t *testing.T) {
	old := PreviewPercent
	defer func() { PreviewPercent = old }()
	PreviewPercent = 63
	if got := Divider(&State{}); got != "change-preview-window(right,63%,border-left,nowrap)" {
		t.Fatalf("root: %q", got)
	}
	st := &State{Stack: []Frame{{Kind: "ai", View: AIView, Title: AITitle}}}
	if got := Divider(st); got != "change-preview-window(right,63%,border-left,wrap,follow)" {
		t.Fatalf("ai pane: %q", got)
	}
	if got := Window(false); got != "right,63%,border-left,nowrap" {
		t.Fatalf("Window: %q", got)
	}
}

func TestRefreshRowInAViewRunsAndReturns(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "view", View: SettingsView, Title: SettingsTitle, Query: "", Pos: 1},
		{Kind: "view", View: "ext/settings/preview", Title: "Preview width", Query: "", Pos: 2},
	}}
	got := Enter(st, "ext/settings/preview=65", "refresh", "65%", "", 3, run)
	if !strings.HasPrefix(got, "execute-silent(swoop-run 'ext/settings/preview=65')+") {
		t.Fatalf("runs the row quietly: %q", got)
	}
	if !strings.Contains(got, "change-prompt(Settings > )") || !strings.Contains(got, "reload-sync(swoop-nav rows {q})") {
		t.Fatalf("returns to the pane below, reloaded: %q", got)
	}
	if len(st.Stack) != 1 || st.Stack[0].View != SettingsView {
		t.Fatalf("stack: %+v", st.Stack)
	}
	// At the root a refresh row is just a row: it runs and the launcher ends.
	if got := Enter(&State{}, "ext/x/y", "refresh", "y", "", 1, run); !strings.HasPrefix(got, "become:") {
		t.Fatalf("at the root: %q", got)
	}
}

func TestSettingsKeyOpensThePaneOnce(t *testing.T) {
	st := &State{}
	got := Settings(st, "typed", 4)
	if !strings.Contains(got, "change-prompt(Settings > )") || len(st.Stack) != 1 || st.Stack[0].Query != "typed" {
		t.Fatalf("got %q, stack %+v", got, st.Stack)
	}
	if got := Settings(st, "", 1); got != "ignore" {
		t.Fatalf("inside the pane: %q", got)
	}
}

func TestToggleRowInAViewRunsAndStays(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "", Pos: 1}}}
	got := Enter(st, "ext/tasks/t\x1fbuy milk", "toggle", "buy milk", "milk", 3, run)
	if !strings.HasPrefix(got, "execute-silent(swoop-run 'ext/tasks/t\x1fbuy milk')+") {
		t.Fatalf("runs the row quietly: %q", got)
	}
	// No pos: fzf keeps the cursor's row number across the reload, and
	// with no wait the next Enter is read, not dropped.
	if !strings.HasSuffix(got, "+clear-query+reload-sync(swoop-nav rows {q})") || st.Land != nil {
		t.Fatalf("stays in the pane, reloaded, same row: %q %+v", got, st.Land)
	}
	if len(st.Stack) != 1 {
		t.Fatalf("the pane was popped: %+v", st.Stack)
	}
	// At the root a toggle row is just a row: it runs and the launcher ends.
	if got := Enter(&State{}, "ext/x/y", "toggle", "y", "", 1, run); !strings.HasPrefix(got, "become:") {
		t.Fatalf("at the root: %q", got)
	}
}

func TestGroupRowDoesNothing(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "", Pos: 1}}}
	if got := Enter(st, "ext/tasks/group\x1fToday", "group", "Today", "", 1, run); got != "ignore" {
		t.Fatalf("a header ran: %q", got)
	}
	if len(st.Stack) != 1 || st.Land != nil {
		t.Fatalf("a header moved the stack: %+v", st)
	}
	if got := Enter(&State{}, "ext/x/group", "group", "y", "", 1, run); got != "ignore" {
		t.Fatalf("at the root: %q", got)
	}
}

func TestTerminalRowHandsOverTheTerminalAndStays(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/notes/notes", Title: "Notes", Query: "", Pos: 1}}}
	got := Enter(st, "ext/notes/fake.md", "terminal", "Fake (draft)", "zuc", 2, run)
	want := "execute[env SWOOP_KIND='terminal' SWOOP_TITLE='Fake (draft)' swoop-run 'ext/notes/fake.md']" +
		"+transform(swoop-nav back 'ext/notes/fake.md')"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if strings.Contains(got, "cleanup") || strings.Contains(got, "clear-query") {
		t.Fatalf("a terminal row must not end the launcher or clear the bar: %q", got)
	}
	if len(st.Stack) != 1 {
		t.Fatalf("the pane was popped: %+v", st.Stack)
	}
	// At the root too: the launcher stays open.
	if got := Enter(&State{}, "ext/x/y", "terminal", "y", "", 1, run); !strings.HasPrefix(got, "execute(") {
		t.Fatalf("at the root: %q", got)
	}
}

func TestTerminalActionPopsBackToThePaneBelow(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "", Pos: 1},
		{Kind: "actions", View: "ext/tasks/t\x1fmilk", Title: "milk", Query: "mi", Pos: 3},
	}}
	got := Enter(st, "edit", "terminal", "Edit the list", "", 1, run)
	if !strings.HasPrefix(got, "execute(swoop-run 'ext/tasks/t\x1fmilk' 'edit')+enable-search") && !strings.HasPrefix(got, "execute(swoop-run 'ext/tasks/t\x1fmilk' 'edit')+disable-search") {
		t.Fatalf("runs the action with the terminal: %q", got)
	}
	if !strings.HasSuffix(got, "change-query(mi)+reload-sync(swoop-nav rows {q})+rebind(result-final)") {
		t.Fatalf("back to the pane below, reloaded: %q", got)
	}
	if st.Land == nil || *st.Land != (Landing{Query: "mi", Pos: 3, Refresh: true}) {
		t.Fatalf("back on row 3, the preview redrawn: %+v", st.Land)
	}
	if strings.Contains(got, "cleanup") {
		t.Fatalf("a terminal action must not end the launcher: %q", got)
	}
	if len(st.Stack) != 1 {
		t.Fatalf("the actions pane is still there: %+v", st.Stack)
	}
}

func TestLandedPlacesTheCursorOnceAndUnbinds(t *testing.T) {
	st := &State{Land: &Landing{Query: "de", Pos: 3, Refresh: true}}
	if got := Landed(st, "de"); got != "pos(3)+refresh-preview+unbind(result-final)" {
		t.Fatalf("got %q", got)
	}
	if st.Land != nil {
		t.Fatal("a landing happens once")
	}
	if got := Landed(st, "de"); got != "unbind(result-final)" {
		t.Fatalf("nothing pending: %q", got)
	}
	// Typed since the chain: the user has moved on, the cursor stays.
	st.Land = &Landing{Query: "de", Pos: 3}
	if got := Landed(st, "dex"); got != "unbind(result-final)" || st.Land != nil {
		t.Fatalf("typed after: %q %+v", got, st.Land)
	}
}

func TestSettleLandsBeforeAKeyReadsTheRow(t *testing.T) {
	st := &State{}
	if got := Settle(st, "", "swoop-nav enter {1} {2} {4}"); got != "" {
		t.Fatalf("nothing pending, nothing to do: %q", got)
	}
	st.Land = &Landing{Query: "de", Pos: 3}
	got := Settle(st, "de", "swoop-nav enter {1} {2} {4}")
	if got != "pos(3)+unbind(result-final)+transform(swoop-nav enter {1} {2} {4})" {
		t.Fatalf("got %q", got)
	}
	if st.Land != nil {
		t.Fatal("settled means landed")
	}
}

// TestNoChainWaits is the record of which chains wait: none. fzf drops
// keys during wait, so a chain that must put the cursor on a new list
// arms the landing event instead (see LandEvent). The one wait left is
// in bin/swoop, at the head of the keys that read the row under the
// cursor, and it holds only while a list is on its way.
func TestNoChainWaits(t *testing.T) {
	view := func() *State {
		return &State{Stack: []Frame{
			{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "mi", Pos: 3},
			{Kind: "actions", View: "ext/tasks/t", Title: "milk", Query: "", Pos: 1},
		}}
	}
	inView := func() *State { return &State{Stack: view().Stack[:1]} }
	ai := &State{Stack: []Frame{{Kind: "ai", View: AIView, Title: AITitle}}}
	chains := map[string]string{
		"pop":             Esc(inView(), ""),
		"pop actions":     Esc(view(), ""),
		"refresh action":  Enter(view(), "delete", "refresh", "Delete", "", 1, run),
		"terminal action": Enter(view(), "edit", "terminal", "Edit", "", 1, run),
		"refresh row":     Enter(&State{Stack: view().Stack[:1]}, "x", "refresh", "x", "", 1, run),
		"toggle":          Enter(inView(), "t", "toggle", "t", "", 2, run),
		"terminal row":    Enter(inView(), "n", "terminal", "n", "", 2, run),
		"push":            Enter(&State{}, "v", "view", "V", "", 1, run),
		"actions":         Actions(&State{}, "v", "view", "V", "", 1),
		"ask":             Ask(&State{}, "", 1),
		"send":            AfterSend(ai),
		"back":            Back(inView(), ""),
		"back, landing":   Back(inView(), "ext/notes/new.md"),
	}
	for name, got := range chains {
		for _, a := range strings.Split(got, "+") {
			if a == "wait" {
				t.Errorf("%s waits, and would drop the keys typed meanwhile: %q", name, got)
			}
		}
	}
}

func TestBackKeepsTheBarUnlessTheRunNamesARow(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/notes/notes", Title: "Notes"}}}
	// An edit: the bar as it was, the same row number, the preview redrawn.
	if got := Back(st, ""); got != "reload-sync(swoop-nav rows {q})+refresh-preview" || st.Land != nil {
		t.Fatalf("got %q %+v", got, st.Land)
	}
	// A New note: the bar cleared, the cursor on the note it made.
	if got := Back(st, "ext/notes/fake-idea.md"); got != "clear-query+reload-sync(swoop-nav rows {q})+rebind(result-final)" {
		t.Fatalf("got %q", got)
	}
	if st.Land == nil || *st.Land != (Landing{ID: "ext/notes/fake-idea.md"}) {
		t.Fatalf("land on the note: %+v", st.Land)
	}
	ids := []string{"ext/notes/+new", "ext/notes/fake-idea.md", "ext/notes/fake-meeting.md"}
	if Find(st, "typed", ids) {
		t.Fatal("rows for other text are not the rows the landing waits on")
	}
	if !Find(st, "", ids) || st.Land.Pos != 2 {
		t.Fatalf("the note is row 2: %+v", st.Land)
	}
	if Find(st, "", ids) {
		t.Fatal("found once; later reloads leave it")
	}
	if got := Landed(st, ""); got != "pos(2)+unbind(result-final)" {
		t.Fatalf("got %q", got)
	}
}
