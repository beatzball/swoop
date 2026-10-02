package nav

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/protocol"
)

// A view whose bar is a prompt and whose preview is a transcript's, and a
// plain view a key opens: what two bundled extensions ask for in their
// views and key files. The rules know neither by name.
const (
	askView      = "ext/ai/ask"
	askTitle     = "Ask AI"
	settingsView = "ext/settings/settings"
)

var askPane = ext.Pane{Prompt: true, Wrap: true, Follow: true}

// inAsk is the stack inside that prompt pane.
func inAsk(query string, pos int) *State {
	return &State{Stack: []Frame{{Kind: "view", View: askView, Title: askTitle, Query: query, Pos: pos, Pane: askPane}}}
}

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
	got := Enter(st, "/Applications/Safari.app", "app", "Safari", "saf", 1, ext.Pane{}, run)
	if got != "become:cleanup; swoop-run '/Applications/Safari.app'" {
		t.Fatalf("got %q", got)
	}
	if len(st.Stack) != 0 {
		t.Fatal("an action row must not push a pane")
	}
}

func TestEnterOnNothingIsIgnored(t *testing.T) {
	if got := Enter(&State{}, "", "", "", "zzz", 0, ext.Pane{}, run); got != "ignore" {
		t.Fatalf("got %q", got)
	}
}

func TestEnterOnViewRowPushesAndLoads(t *testing.T) {
	st := &State{}
	got := Enter(st, "ext/define/define", "view", "Define Word", "de", 3, ext.Pane{}, run)
	want := "clear-query+disable-search+change-prompt(Define Word > )+reload-sync(swoop-nav rows {q})+first+rebind(result-final)"
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
	want := "clear-query+disable-search+change-prompt(hello actions > )+reload-sync(swoop-nav rows {q})+first+rebind(result-final)+change-preview(swoop-preview 'ext/clipboard/17')"
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
	got := Enter(st, "delete", "refresh", "Delete", "", 2, ext.Pane{}, run)
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
	got := Enter(st, "reveal", "action", "Reveal in Finder", "", 2, ext.Pane{}, run)
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
	want := "change-query(ap)+disable-search+change-prompt(Define Word > )+reload-sync(swoop-nav rows {q})+first+rebind(result-final)"
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

func TestKeywordOpensAPromptPaneWithTheRest(t *testing.T) {
	st := &State{}
	got := Change(st, &Keyed{Rest: "why is the sky blue", View: askView, Title: askTitle, Pane: askPane})
	want := "change-query(why is the sky blue)+disable-search+change-prompt(Ask AI > )+change-preview-window(right,58%,border-left,wrap,follow)+reload-sync(swoop-nav rows)+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || !st.Stack[0].Pane.Prompt || st.Stack[0].Query != "" {
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

func TestAKeyOpensAPromptPaneAndKeepsTheText(t *testing.T) {
	st := &State{}
	got := Key(st, askView, askTitle, askPane, "why is the sky blue", 3)
	want := "disable-search+change-prompt(Ask AI > )+change-preview-window(right,58%,border-left,wrap,follow)+reload-sync(swoop-nav rows)+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(st.Stack) != 1 || st.Stack[0].Kind != "view" || st.Stack[0].View != askView || st.Stack[0].Pane != askPane || st.Stack[0].Query != "why is the sky blue" || st.Stack[0].Pos != 3 {
		t.Fatalf("stack: %+v", st.Stack)
	}
	if got := Key(st, askView, askTitle, askPane, "", 1); got != "ignore" {
		t.Fatalf("the key inside the pane does nothing: got %q", got)
	}
	if got := Change(st, nil); got != "ignore" {
		t.Fatalf("typing in the pane must not reload: got %q", got)
	}
}

func TestAKeyOnAnEmptyBarOpensThePaneToo(t *testing.T) {
	st := &State{}
	if got := Key(st, askView, askTitle, askPane, "", 1); got == "ignore" {
		t.Fatal("an empty bar still opens the pane")
	}
	if len(st.Stack) != 1 || st.Stack[0].Query != "" {
		t.Fatalf("stack: %+v", st.Stack)
	}
}

func TestEnterInAPromptPaneSendsThenClears(t *testing.T) {
	st := inAsk("", 0)
	if Sends(st, "ext/ai/new", "  ") {
		t.Fatal("nothing to send")
	}
	if Sends(st, "", "why") {
		t.Fatal("no row, nothing to send to")
	}
	if !Sends(st, "ext/ai/new", "why") || !Sends(st, "ext/ai/20260925-1", "and then") {
		t.Fatal("text and a row: the text goes to the row")
	}
	if Sends(&State{}, "ext/ai/new", "why") {
		t.Fatal("only inside the pane")
	}
	if Sends(&State{Stack: []Frame{{Kind: "view", View: "ext/define/define", Title: "Define Word"}}}, "ext/define/apple", "why") {
		t.Fatal("only where the bar is a prompt")
	}
	// The send named the row it wrote to: the cursor goes there.
	if got := AfterSend(st, "ext/ai/20260925-1", 1); got != "clear-query+reload-sync(swoop-nav rows)+rebind(result-final)" {
		t.Fatalf("got %q", got)
	}
	if st.Land == nil || *st.Land != (Landing{ID: "ext/ai/20260925-1", Refresh: true}) {
		t.Fatalf("the cursor goes to the conversation once the list lands: %+v", st.Land)
	}
	if !Find(st, "", []string{"ext/ai/new", "ext/ai/20260925-1"}) || st.Land.Pos != 2 {
		t.Fatalf("the conversation is row 2: %+v", st.Land)
	}
	// The pane's own worker reloads it, so the event does not stay bound.
	if got := Landed(st, "", "conversation", 1); got != "pos(2)+transform(swoop-nav step down {2} 0)+refresh-preview+unbind(result-final)" {
		t.Fatalf("got %q", got)
	}
	// It named none: the row number it was on, redrawn.
	AfterSend(st, "", 3)
	if st.Land == nil || *st.Land != (Landing{Pos: 3, Refresh: true}) {
		t.Fatalf("the cursor stays on its row: %+v", st.Land)
	}
	st.Land = nil
	if got := Enter(st, "ext/ai/20260925-1", "conversation", "earlier", "and then", 2, ext.Pane{}, run); got != "ignore" {
		t.Fatalf("Enter itself neither runs nor pushes in the pane: %q", got)
	}
	if len(st.Stack) != 1 {
		t.Fatalf("Enter must not push: %+v", st.Stack)
	}
}

func TestEscFromAPromptPaneRestoresTheBar(t *testing.T) {
	st := inAsk("why is the sky blue", 2)
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

func TestPoppingActionsInsideAPaneKeepsItsWindow(t *testing.T) {
	st := &State{Stack: []Frame{
		inAsk("", 1).Stack[0],
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
	st := inAsk("", 0)
	if got := Divider(st); got != "change-preview-window(right,63%,border-left,wrap,follow)" {
		t.Fatalf("a wrapped pane: %q", got)
	}
	// An actions pane keeps the window of the pane it was opened from.
	st.Stack = append(st.Stack, Frame{Kind: "actions", View: "ext/ai/new", Title: "New conversation"})
	if got := Divider(st); got != "change-preview-window(right,63%,border-left,wrap,follow)" {
		t.Fatalf("its actions pane: %q", got)
	}
	if got := Window(ext.Pane{}); got != "right,63%,border-left,nowrap" {
		t.Fatalf("Window: %q", got)
	}
}

// A view's own preview width is set on the way in, from a key or from
// Enter on its row, and the user's is put back on the way out.
func TestAViewsOwnPreviewWidth(t *testing.T) {
	wide := ext.Pane{Percent: 70}
	st := &State{}
	got := Enter(st, "ext/fake/wide", "view", "Wide", "wi", 2, wide, run)
	want := "clear-query+disable-search+change-prompt(Wide > )+change-preview-window(right,70%,border-left,nowrap)+reload-sync(swoop-nav rows {q})+first+rebind(result-final)"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if !OwnWidth(st) || OwnWidth(&State{}) || OwnWidth(inAsk("", 0)) {
		t.Fatal("only the pane with a width of its own")
	}
	if got := Divider(st); got != "change-preview-window(right,70%,border-left,nowrap)" {
		t.Fatalf("the divider leaves it: %q", got)
	}
	// A plain view opened from it gets the plain window back, and a pop
	// to it the wide one.
	got = Enter(st, "ext/fake/inner", "view", "Inner", "", 1, ext.Pane{}, run)
	if !strings.Contains(got, "change-preview-window(right,58%,border-left,nowrap)") {
		t.Fatalf("a plain view over it: %q", got)
	}
	if got := Esc(st, ""); !strings.Contains(got, "change-preview-window(right,70%,border-left,nowrap)") {
		t.Fatalf("back to it: %q", got)
	}
	if got := Esc(st, ""); !strings.Contains(got, "change-preview-window(right,58%,border-left,nowrap)") {
		t.Fatalf("back to the root: %q", got)
	}
}

// Enter on a view row whose view is a prompt pane opens it with the bar
// empty: the text there found the row, and is not a prompt.
func TestEnterOnARowOpensAPromptPaneEmpty(t *testing.T) {
	st := &State{}
	got := Enter(st, askView, "view", askTitle, "ask", 2, askPane, run)
	want := "clear-query+disable-search+change-prompt(Ask AI > )+change-preview-window(right,58%,border-left,wrap,follow)+reload-sync(swoop-nav rows)+first"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestRefreshRowInAViewRunsAndReturns(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "view", View: settingsView, Title: "Settings", Query: "", Pos: 1},
		{Kind: "view", View: "ext/settings/preview", Title: "Preview width", Query: "", Pos: 2},
	}}
	got := Enter(st, "ext/settings/preview=65", "refresh", "65%", "", 3, ext.Pane{}, run)
	if !strings.HasPrefix(got, "execute-silent(swoop-run 'ext/settings/preview=65')+") {
		t.Fatalf("runs the row quietly: %q", got)
	}
	if !strings.Contains(got, "change-prompt(Settings > )") || !strings.Contains(got, "reload-sync(swoop-nav rows {q})") {
		t.Fatalf("returns to the pane below, reloaded: %q", got)
	}
	if len(st.Stack) != 1 || st.Stack[0].View != settingsView {
		t.Fatalf("stack: %+v", st.Stack)
	}
	// At the root a refresh row is just a row: it runs and the launcher ends.
	if got := Enter(&State{}, "ext/x/y", "refresh", "y", "", 1, ext.Pane{}, run); !strings.HasPrefix(got, "become:") {
		t.Fatalf("at the root: %q", got)
	}
}

func TestAKeyOpensAPlainViewOnce(t *testing.T) {
	st := &State{}
	got := Key(st, settingsView, "Settings", ext.Pane{}, "typed", 4)
	// As Enter on its row: the bar cleared, and no change of window.
	if got != "clear-query+disable-search+change-prompt(Settings > )+reload-sync(swoop-nav rows {q})+first+rebind(result-final)" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "change-prompt(Settings > )") || len(st.Stack) != 1 || st.Stack[0].Query != "typed" {
		t.Fatalf("got %q, stack %+v", got, st.Stack)
	}
	if got := Key(st, settingsView, "Settings", ext.Pane{}, "", 1); got != "ignore" {
		t.Fatalf("inside the pane: %q", got)
	}
}

func TestToggleRowInAViewRunsAndStays(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "", Pos: 1}}}
	got := Enter(st, "ext/tasks/t\x1fbuy milk", "toggle", "buy milk", "milk", 3, ext.Pane{}, run)
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
	if got := Enter(&State{}, "ext/x/y", "toggle", "y", "", 1, ext.Pane{}, run); !strings.HasPrefix(got, "become:") {
		t.Fatalf("at the root: %q", got)
	}
}

// Enter reaches a header only on the second try (see OffHeader), and
// then does nothing.
func TestGroupRowDoesNothing(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "", Pos: 1}}}
	if got := Enter(st, "ext/tasks/group\x1fToday", "group", "Today", "", 1, ext.Pane{}, run); got != "ignore" {
		t.Fatalf("a header ran: %q", got)
	}
	if len(st.Stack) != 1 || st.Land != nil {
		t.Fatalf("a header moved the stack: %+v", st)
	}
	if got := Enter(&State{}, "ext/x/group", "group", "y", "", 1, ext.Pane{}, run); got != "ignore" {
		t.Fatalf("at the root: %q", got)
	}
}

func TestTerminalRowHandsOverTheTerminalAndStays(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/notes/notes", Title: "Notes", Query: "", Pos: 1}}}
	got := Enter(st, "ext/notes/fake.md", "terminal", "Fake (draft)", "zuc", 2, ext.Pane{}, run)
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
	if got := Enter(&State{}, "ext/x/y", "terminal", "y", "", 1, ext.Pane{}, run); !strings.HasPrefix(got, "execute(") {
		t.Fatalf("at the root: %q", got)
	}
}

func TestTerminalActionPopsBackToThePaneBelow(t *testing.T) {
	st := &State{Stack: []Frame{
		{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks", Query: "", Pos: 1},
		{Kind: "actions", View: "ext/tasks/t\x1fmilk", Title: "milk", Query: "mi", Pos: 3},
	}}
	got := Enter(st, "edit", "terminal", "Edit the list", "", 1, ext.Pane{}, run)
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
	// The row landed on may be a header, so a step follows the pos.
	if got := Landed(st, "de", "app", 1); got != "pos(3)+transform(swoop-nav step down {2} 0)+refresh-preview+unbind(result-final)" {
		t.Fatalf("got %q", got)
	}
	if st.Land != nil {
		t.Fatal("a landing happens once")
	}
	if got := Landed(st, "de", "app", 3); got != "unbind(result-final)" || st.Rest != 3 {
		t.Fatalf("nothing pending: %q, rest %d", got, st.Rest)
	}
	// Typed since the chain: the user has moved on, the cursor stays.
	st.Land = &Landing{Query: "de", Pos: 3}
	if got := Landed(st, "dex", "app", 1); got != "unbind(result-final)" || st.Land != nil {
		t.Fatalf("typed after: %q %+v", got, st.Land)
	}
}

// Inside a view the event stays bound, and a list that lands with a
// header under the cursor has the cursor stepped off it: a view opened,
// text typed, a tick, a reload the extension asked fzf for.
func TestLandedInAViewStaysBoundAndStepsOffAHeader(t *testing.T) {
	st := &State{Stack: []Frame{{Kind: "view", View: "ext/tasks/tasks", Title: "Tasks"}}}
	if got := Landed(st, "", "group", 1); got != "transform(swoop-nav step down {2} 0)" || st.Rest != 0 {
		t.Fatalf("a header under the cursor: %q, rest %d", got, st.Rest)
	}
	if got := Landed(st, "", "toggle", 2); got != "" || st.Rest != 2 {
		t.Fatalf("a task under the cursor, nothing to do: %q, rest %d", got, st.Rest)
	}
	st.Land = &Landing{Pos: 4}
	if got := Landed(st, "", "toggle", 2); got != "pos(4)+transform(swoop-nav step down {2} 0)" {
		t.Fatalf("a landing in a view: %q", got)
	}
	// An actions pane on top of the view is not a view.
	st.Stack = append(st.Stack, Frame{Kind: "actions", View: "ext/tasks/t"})
	if got := Landed(st, "", "action", 1); got != "unbind(result-final)" {
		t.Fatalf("in an actions pane: %q", got)
	}
}

func TestStepGoesOverAHeaderTheSameWay(t *testing.T) {
	st := &State{}
	// Down from the last task of a group comes to the next group's header.
	if got := Step(st, "down", "group", 3, 0, false); got != "down+transform(swoop-nav step down {2} 3)" {
		t.Fatalf("down: %q", got)
	}
	if got := Step(st, "up", "group", 3, 0, false); got != "up+transform(swoop-nav step up {2} 3)" {
		t.Fatalf("up: %q", got)
	}
	if st.Rest != 0 {
		t.Fatalf("a header is not a place to rest: %d", st.Rest)
	}
	// On a task there is nothing to do but remember the row.
	if got := Step(st, "down", "toggle", 4, 3, false); got != "" || st.Rest != 4 {
		t.Fatalf("on a task: %q, rest %d", got, st.Rest)
	}
	// No row at all, an empty list: nothing, and nothing remembered.
	if got := Step(st, "down", "", 0, 0, false); got != "" || st.Rest != 4 {
		t.Fatalf("no row: %q, rest %d", got, st.Rest)
	}
	// A landing in flight owns the state file's cursor.
	st.Land = &Landing{ID: "x"}
	if Step(st, "down", "toggle", 5, 0, false); st.Rest != 4 {
		t.Fatalf("remembered during a landing: %d", st.Rest)
	}
}

func TestStepTurnsRoundAtAnEndOnce(t *testing.T) {
	st := &State{}
	// Up from the first task: the header above it is the first row.
	if got := Step(st, "up", "group", 1, 0, false); got != "down+transform(swoop-nav step down {2} 1 turned)" {
		t.Fatalf("the first row: %q", got)
	}
	// Down onto a header that is the last row: the move did nothing.
	if got := Step(st, "down", "group", 9, 9, false); got != "up+transform(swoop-nav step up {2} 9 turned)" {
		t.Fatalf("the last row: %q", got)
	}
	// Headers only: the second end is the end.
	if got := Step(st, "up", "group", 1, 2, true); got != "" {
		t.Fatalf("turned twice: %q", got)
	}
	if got := Step(st, "down", "group", 9, 9, true); got != "" {
		t.Fatalf("turned twice: %q", got)
	}
}

func TestOffHeaderMovesThenDoesTheKeyAgain(t *testing.T) {
	got := OffHeader("swoop-nav enter {1} {2} {4} again")
	if got != "transform(swoop-nav step down {2} 0)+transform(swoop-nav enter {1} {2} {4} again)" {
		t.Fatalf("got %q", got)
	}
}

func TestClickOnAHeaderPutsTheCursorBack(t *testing.T) {
	if got := Click(&State{Rest: 4}); got != "pos(4)+transform(swoop-nav step down {2} 0)" {
		t.Fatalf("got %q", got)
	}
	if got := Click(&State{}); got != "transform(swoop-nav step down {2} 0)" {
		t.Fatalf("no row remembered: %q", got)
	}
}

func TestHeadersPutsTheTitleInTheIconColumn(t *testing.T) {
	items := []protocol.Item{
		{ID: "g", Kind: "group", Title: "Today", Subtitle: "2 tasks"},
		{ID: "t", Kind: "toggle", Icon: "☐", Title: "fake task"},
	}
	Headers(items)
	if want := (protocol.Item{ID: "g", Kind: "group", Icon: "\x1b[2mToday\x1b[22m", Subtitle: "2 tasks"}); items[0] != want {
		t.Fatalf("the header: %+v", items[0])
	}
	if want := (protocol.Item{ID: "t", Kind: "toggle", Icon: "☐", Title: "fake task"}); items[1] != want {
		t.Fatalf("a task is left as it is: %+v", items[1])
	}
}

func TestSettleLandsBeforeAKeyReadsTheRow(t *testing.T) {
	st := &State{}
	if got := Settle(st, "", "swoop-nav enter {1} {2} {4}"); got != "" {
		t.Fatalf("nothing pending, nothing to do: %q", got)
	}
	st.Land = &Landing{Query: "de", Pos: 3}
	got := Settle(st, "de", "swoop-nav enter {1} {2} {4}")
	if got != "pos(3)+transform(swoop-nav step down {2} 0)+unbind(result-final)+transform(swoop-nav enter {1} {2} {4})" {
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
	chains := map[string]string{
		"pop":             Esc(inView(), ""),
		"pop actions":     Esc(view(), ""),
		"refresh action":  Enter(view(), "delete", "refresh", "Delete", "", 1, ext.Pane{}, run),
		"terminal action": Enter(view(), "edit", "terminal", "Edit", "", 1, ext.Pane{}, run),
		"refresh row":     Enter(&State{Stack: view().Stack[:1]}, "x", "refresh", "x", "", 1, ext.Pane{}, run),
		"toggle":          Enter(inView(), "t", "toggle", "t", "", 2, ext.Pane{}, run),
		"terminal row":    Enter(inView(), "n", "terminal", "n", "", 2, ext.Pane{}, run),
		"push":            Enter(&State{}, "v", "view", "V", "", 1, ext.Pane{}, run),
		"actions":         Actions(&State{}, "v", "view", "V", "", 1),
		"key":             Key(&State{}, settingsView, "Settings", ext.Pane{}, "", 1),
		"key, a prompt":   Key(&State{}, askView, askTitle, askPane, "", 1),
		"send":            AfterSend(inAsk("", 0), "ext/ai/1", 2),
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
	if got := Landed(st, "", "note", 1); got != "pos(2)+transform(swoop-nav step down {2} 0)" {
		t.Fatalf("got %q", got)
	}
}
