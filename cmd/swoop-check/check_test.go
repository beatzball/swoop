package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// patient is the limit the fixtures get: the contract's two seconds are
// for a launcher that must not wait, and a machine busy compiling has
// tripped them. The cases about time set their own.
const patient = 10 * time.Second

// good is an extension that meets the contract and uses each part of it:
// a plain row, a view with a header and a toggle, a menu of the three
// kinds, and a view whose bar is a prompt, opened by a key. Every case
// below is this one with one thing broken. It is the text the loader
// runs, not a program of its own.
const good = `row() { printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" '' "$3" "$4"; }
case "${1:-}" in
  list)
    row one text 'One' 'a plain row'
    row pane view 'Pane' 'opens a view'
    ;;
  view)
    case "${2:-}" in
      pane)
        row today group 'Today' '1 task'
        row tick toggle 'Tick' ''
        ;;
      ask)
        row new chat 'New conversation' ''
        ;;
      *) exit 2 ;;
    esac
    ;;
  preview) echo "the preview of ${2:-}" ;;
  actions)
    row copy action 'Copy' ''
    row delete refresh 'Delete' ''
    row edit terminal 'Edit' ''
    ;;
  run) : ;;
  send) : ;;
  *) exit 2 ;;
esac
`

// goodFiles are the files beside it.
var goodFiles = map[string]string{
	"keyword": "fx\n",
	"key":     "# the key that opens the prompt\ntab ask Ask\n",
	"views":   "ask bar=prompt preview=40%,wrap,follow\n",
}

// loader is the program of every fixture: it runs the text in the file
// called body beside it. It is one file, linked into each fixture's
// folder, and not a script written out per fixture, because macOS looks
// a new program over the first time it is run, a quarter of a second
// each, and forty fixtures were ten seconds of that. A link is the same
// file to it.
const loader = "#!/bin/bash\n. \"${0%/*}/body\"\n"

// loaderPath is the one copy, made by TestMain.
var loaderPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "swoop-check-test")
	if err != nil {
		panic(err)
	}
	loaderPath = filepath.Join(dir, "loader")
	if err := os.WriteFile(loaderPath, []byte(loader), 0o755); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// fixture writes an extension called fx into a fresh folder and returns
// the path of its folder: the loader as its program, body as what it
// runs. With no body the folder has no program. A file whose text is ""
// is left out.
func fixture(t *testing.T, body string, files map[string]string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bash scripts do not run as programs on Windows")
	}
	dir := filepath.Join(t.TempDir(), "fx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		// A copy where a link cannot be made, across two file systems.
		if err := os.Link(loaderPath, filepath.Join(dir, "fx")); err != nil {
			if err := os.WriteFile(filepath.Join(dir, "fx"), []byte(loader), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "body"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range files {
		if text == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// swap is good with old replaced by new. old must be there, so a case
// cannot pass because its edit missed.
func swap(t *testing.T, old, new string) string {
	t.Helper()
	if !strings.Contains(good, old) {
		t.Fatalf("the fixture has no %q to break", old)
	}
	return strings.Replace(good, old, new, 1)
}

// with is goodFiles with one file changed.
func with(name, text string) map[string]string {
	files := map[string]string{}
	for k, v := range goodFiles {
		files[k] = v
	}
	files[name] = text
	return files
}

func rulesOf(found []finding) []string {
	var names []string
	for _, f := range found {
		if !slices.Contains(names, f.rule) {
			names = append(names, f.rule)
		}
	}
	return names
}

func TestGoodExtensionMeetsTheContract(t *testing.T) {
	dir := fixture(t, good, goodFiles)
	// With -act: every verb is asked.
	name, found, untried := check(dir, patient, true)
	if name != "fx" || len(found) != 0 || len(untried) != 0 {
		t.Fatalf("check -act = %q, %v, %v; want fx and nothing found, nothing left", name, found, untried)
	}
	// Without: the same, and it says what it left alone.
	_, found, untried = check(dir, patient, false)
	if len(found) != 0 || !slices.Equal(untried, []string{"run", "send"}) {
		t.Fatalf("check = %v, %v; want nothing found, run and send not tried", found, untried)
	}
	// Named by its program, it is the same extension.
	if name, found, _ := check(filepath.Join(dir, "fx"), patient, false); name != "fx" || len(found) != 0 {
		t.Fatalf("check by program = %q, %v", name, found)
	}
}

// One fixture per way to break each rule. Each must be found, under that
// rule and no other: a finding that names the wrong rule sends its
// reader to the wrong part of the contract.
func TestEachBrokenRuleIsNamed(t *testing.T) {
	type broken struct {
		name  string
		rule  string
		body  func(t *testing.T) string
		files map[string]string
		limit time.Duration
		// quiet is a case -act would hide: its run or send is never
		// reached, or must not be asked.
		quiet bool
	}
	same := func(t *testing.T) string { return good }
	cases := []broken{
		{name: "no program in the folder", rule: ruleProgram, body: func(t *testing.T) string { return "" }},

		{name: "list fails", rule: ruleList, body: func(t *testing.T) string {
			return swap(t, "  list)\n", "  list)\n    echo 'no network' >&2; exit 3\n")
		}},
		{name: "list fails with a text", rule: ruleList, body: func(t *testing.T) string {
			return swap(t, "  list)\n", "  list)\n    [ -z \"${2:-}\" ] || exit 3\n")
		}},

		{name: "a view row's view fails", rule: ruleView, body: func(t *testing.T) string {
			return swap(t, "      pane)\n", "      pane)\n        exit 4\n")
		}},
		{name: "a key names a view that does not answer", rule: ruleView, body: same, files: with("key", "tab nowhere Nowhere\n")},
		{name: "views names a view that does not answer", rule: ruleView, body: same, files: with("views", "ask bar=prompt\nnowhere preview=wrap\n")},

		{name: "preview fails for an id it printed", rule: rulePreview, body: func(t *testing.T) string {
			return swap(t, `preview) echo "the preview of ${2:-}" ;;`, `preview) [ "${2:-}" != tick ] || exit 1 ;;`)
		}},

		{name: "run fails for an id it printed", rule: ruleRun, body: func(t *testing.T) string {
			return swap(t, "  run) : ;;", "  run) exit 1 ;;")
		}},

		{name: "an action of a kind that is not one", rule: ruleActions, body: func(t *testing.T) string {
			return swap(t, "row delete refresh", "row delete toggle")
		}},

		{name: "a prompt view with no send", rule: ruleSend, body: func(t *testing.T) string {
			return swap(t, "  send) : ;;\n", "")
		}},
		{name: "a prompt view with no row", rule: ruleSend, quiet: true, body: func(t *testing.T) string {
			return swap(t, "        row new chat 'New conversation' ''\n", "        :\n")
		}},

		{name: "list is not done in time", rule: ruleTime, limit: 300 * time.Millisecond, quiet: true, body: func(t *testing.T) string {
			return swap(t, "  list)\n", "  list)\n    exec sleep 5\n")
		}},
		{name: "view is not done in time", rule: ruleTime, limit: 300 * time.Millisecond, quiet: true, body: func(t *testing.T) string {
			return swap(t, "      pane)\n", "      pane)\n        exec sleep 5\n")
		}},
		{name: "actions is not done in time", rule: ruleTime, limit: 300 * time.Millisecond, quiet: true, body: func(t *testing.T) string {
			return swap(t, "  actions)\n", "  actions)\n    exec sleep 5\n")
		}},
		{name: "send is not done in time", rule: ruleTime, limit: 300 * time.Millisecond, body: func(t *testing.T) string {
			return swap(t, "  send) : ;;", "  send) exec sleep 5 ;;")
		}},
		{name: "list leaves a worker holding its output", rule: ruleTime, body: func(t *testing.T) string {
			return swap(t, "  list)\n", "  list)\n    sleep 3 &\n")
		}},

		{name: "a line with four fields", rule: ruleLine, body: func(t *testing.T) string {
			return swap(t, "    row one text 'One' 'a plain row'\n", "    row one text 'One' 'a plain row'\n    printf 'two\\ttext\\t\\tTwo\\n'\n")
		}},
		{name: "a tab in an id", rule: ruleLine, body: func(t *testing.T) string {
			return swap(t, "    row one text", "    row \"$(printf 'o\\tne')\" text")
		}},
		{name: "an empty id", rule: ruleLine, body: func(t *testing.T) string {
			return swap(t, "    row one text", "    row '' text")
		}},
		{name: "an empty line", rule: ruleLine, body: func(t *testing.T) string {
			return swap(t, "  list)\n", "  list)\n    echo\n")
		}},
		{name: "a line that is not UTF-8", rule: ruleLine, body: func(t *testing.T) string {
			return swap(t, "    row one text 'One'", "    row one text \"$(printf 'On\\377')\"")
		}},
		{name: "an action that is not a row", rule: ruleLine, body: func(t *testing.T) string {
			return swap(t, "  actions)\n", "  actions)\n    echo Copy\n")
		}},

		{name: "two rows with one id", rule: ruleUnique, body: func(t *testing.T) string {
			return swap(t, "    row pane view", "    row one text 'One again' ''\n    row pane view")
		}},

		{name: "a header with no row under it", rule: ruleGroup, body: func(t *testing.T) string {
			return swap(t, "        row tick toggle 'Tick' ''\n", "        row tick toggle 'Tick' ''\n        row later group 'Later' ''\n")
		}},
		{name: "a header over a header", rule: ruleGroup, body: func(t *testing.T) string {
			return swap(t, "        row today group 'Today' '1 task'\n", "        row past group 'Past due' ''\n        row today group 'Today' '1 task'\n")
		}},

		{name: "a keyword of two words", rule: ruleKeyword, body: same, files: with("keyword", "my fx\n")},
		{name: "a keyword file with no word", rule: ruleKeyword, body: same, files: with("keyword", "\n")},

		{name: "a key that may not be claimed", rule: ruleKey, body: same, files: with("key", "enter ask Ask\n")},
		{name: "a key with no view", rule: ruleKey, body: same, files: with("key", "tab\ntab ask Ask\n")},
		{name: "a key claimed twice", rule: ruleKey, body: same, files: with("key", "tab ask Ask\ntab pane Pane\n")},

		{name: "a setting that is misspelled", rule: ruleViews, body: same, files: with("views", "ask bar=promt\n")},
		{name: "a setting the contract has not got", rule: ruleViews, body: same, files: with("views", "ask bar=prompt colour=red\n")},
		{name: "a preview width out of range", rule: ruleViews, body: same, files: with("views", "ask bar=prompt preview=95%\n")},
		{name: "a preview word that is not one", rule: ruleViews, body: same, files: with("views", "ask bar=prompt preview=wide\n")},
		{name: "a view named twice", rule: ruleViews, body: same, files: with("views", "ask bar=prompt\nask preview=wrap\n")},

		{name: "a cache word that is not run", rule: ruleCache, body: same, files: with("cache", "forever\n")},

		{name: "icons with the wrong word", rule: ruleIcons, body: same, files: with("icons", "path\n")},
		{name: "icons with no cache", rule: ruleIcons, body: same, files: with("icons", "id\n")},
	}
	// icons over ids that are not files needs both files at once.
	both := with("cache", "run\n")
	both["icons"] = "id\n"
	cases = append(cases, broken{name: "icons over ids that are no files", rule: ruleIcons, body: same, files: both})

	covered := map[string]bool{}
	for _, c := range cases {
		covered[c.rule] = true
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			files := c.files
			if files == nil {
				files = goodFiles
			}
			limit := c.limit
			if limit == 0 {
				limit = patient
			}
			_, found, _ := check(fixture(t, c.body(t), files), limit, !c.quiet)
			if got := rulesOf(found); !slices.Equal(got, []string{c.rule}) {
				t.Fatalf("found %v; want findings under %q alone", found, c.rule)
			}
		})
	}
	for _, rule := range rules {
		if !covered[rule] {
			t.Errorf("no fixture breaks the rule %q", rule)
		}
	}
}

// An extension with no actions verb has no menu, and that is allowed: the
// smallest extension is list, preview and run.
func TestNoActionsVerbIsNoFinding(t *testing.T) {
	body := swap(t, "  actions)\n    row copy action 'Copy' ''\n    row delete refresh 'Delete' ''\n    row edit terminal 'Edit' ''\n    ;;\n", "")
	if _, found, _ := check(fixture(t, body, goodFiles), patient, true); len(found) != 0 {
		t.Fatalf("found %v; an extension without actions breaks no rule", found)
	}
}

// icons and cache together, over ids that are files, meet the contract.
func TestIconsOverFiles(t *testing.T) {
	// Two rows whose ids are files: the program itself, and its folder.
	body := "case \"${1:-}\" in\n" +
		"  list) printf '%s\\tapp\\t\\tProgram\\t\\n%s\\tapp\\t\\tFolder\\t\\n' \"$SWOOP_EXT_DIR/fx\" \"$SWOOP_EXT_DIR\" ;;\n" +
		"  preview|run) : ;;\n  *) exit 2 ;;\nesac\n"
	dir := fixture(t, body, map[string]string{"cache": "run\n", "icons": "id\n"})
	if _, found, _ := check(dir, patient, false); len(found) != 0 {
		t.Fatalf("found %v; want nothing", found)
	}
}

// What a person reads: a line per finding with the name and the rule
// first, or one line that says the extension passed.
func TestReport(t *testing.T) {
	ok := fixture(t, good, goodFiles)
	bad := fixture(t, swap(t, "    row one text 'One' 'a plain row'\n", "    echo 'not a row'\n"), goodFiles)

	var out bytes.Buffer
	if failed := report(&out, []string{ok, bad}, patient, false); failed != 1 {
		t.Errorf("report says %d failed; want 1", failed)
	}
	want := "fx: meets contract " + contract + " (not tried without -act: run, send)\n" +
		"fx: line: list: line 1 has no tab in it; a row is 5 fields with tabs between\n" +
		"fx: line: list a: line 1 has no tab in it; a row is 5 fields with tabs between\n"
	if out.String() != want {
		t.Errorf("report printed\n%s\nwant\n%s", out.String(), want)
	}
}

// A path that is not there is a finding, not a crash.
func TestMissingPath(t *testing.T) {
	name, found, _ := check(filepath.Join(t.TempDir(), "nothing"), patient, false)
	if name != "nothing" || !slices.Equal(rulesOf(found), []string{ruleProgram}) {
		t.Fatalf("check = %q, %v; want one finding under %q", name, found, ruleProgram)
	}
}

// Many bad lines are a few findings and a count, not a screen of them.
func TestManyBadLinesAreCounted(t *testing.T) {
	body := swap(t, "  list)\n", "  list)\n    for i in 1 2 3 4 5 6; do echo \"bad $i\"; done\n")
	_, found, _ := check(fixture(t, body, goodFiles), patient, false)
	// list and list <text>: each maxShown lines and one count.
	if len(found) != 2*(maxShown+1) {
		t.Fatalf("found %d findings, want %d: %v", len(found), 2*(maxShown+1), found)
	}
	if last := found[maxShown].text; !strings.Contains(last, "3 more lines") {
		t.Errorf("the count line says %q", last)
	}
}
