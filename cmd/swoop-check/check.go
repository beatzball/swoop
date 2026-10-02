package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/beatzball/swoop/internal/ext"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
)

// contract is the version of CONTRACT.md this program checks against. A
// test holds the two to the same number.
const contract = "1"

// The rules, by the name a finding carries. CONTRACT.md lists the same
// names with what each asks, and a test holds the two lists together.
const (
	ruleProgram = "program" // the folder holds a program of its own name
	ruleList    = "list"    // list exits 0, with a text and without
	ruleView    = "view"    // view exits 0 for every view the extension names
	rulePreview = "preview" // preview exits 0 for an id the program printed
	ruleRun     = "run"     // run exits 0 for an id the program printed
	ruleActions = "actions" // an actions row is an action, a refresh or a terminal
	ruleSend    = "send"    // a view whose bar is a prompt has a row, and takes send
	ruleTime    = "time"    // list, view, actions and send keep the limit
	ruleLine    = "line"    // a line is five tab-separated UTF-8 fields, the id not empty
	ruleUnique  = "unique"  // no two rows of one list share an id
	ruleGroup   = "group"   // a header has a row under it
	ruleKeyword = "keyword" // the keyword file holds one word
	ruleKey     = "key"     // a key file's line is a key that may be claimed and a view
	ruleViews   = "views"   // a views file's line is a view and settings the contract has
	ruleCache   = "cache"   // the cache file holds the word run
	ruleIcons   = "icons"   // the icons file holds the word id, beside cache, over ids that are files
)

var rules = []string{
	ruleProgram, ruleList, ruleView, rulePreview, ruleRun, ruleActions,
	ruleSend, ruleTime, ruleLine, ruleUnique, ruleGroup, ruleKeyword,
	ruleKey, ruleViews, ruleCache, ruleIcons,
}

// guard is how long a verb with no limit in the contract, preview and
// run, may take before the checker gives up on it. Not a rule of the
// launcher's: it only keeps a stuck program from holding the checker.
const guard = 10 * time.Second

// maxLine is the longest line the launcher reads. Past it, the launcher
// stops reading the list: that line and every row after it are lost.
const maxLine = 1 << 20

// maxViews is how many views of one extension are opened. A view may
// list views that list views; the checker is not there to walk a tree.
const maxViews = 8

// maxShown is how many findings of one sort, about one list, are printed.
// A program that prints four fields prints them on every line.
const maxShown = 3

// probe is the text a list and a view are asked with, as if typed.
const probe = "a"

// finding is one place the extension does not meet the contract.
type finding struct {
	rule, text string
}

type checker struct {
	e     ext.Extension
	limit time.Duration
	act   bool
	env   []string
	found []finding
	// untried names the verbs left alone for want of -act.
	untried []string
	// runnable is the first row found that Enter would run and leave.
	runnable string
	// queue is the views still to open, seen the ones opened or queued.
	queue []string
	seen  map[string]bool
}

func (c *checker) fail(rule, format string, a ...any) {
	c.found = append(c.found, finding{rule, fmt.Sprintf(format, a...)})
}

// check runs the extension at path, its folder or its program, against
// the contract. It returns the extension's name, what it found, and the
// verbs it did not try.
func check(path string, limit time.Duration, act bool) (name string, found []finding, untried []string) {
	e, err := find(path)
	if err != nil {
		return e.Name, []finding{{ruleProgram, err.Error()}}, nil
	}
	// A land file, as the launcher gives every run: a program may write
	// there, and one written under `set -u` reads the variable.
	tmp, err := os.MkdirTemp("", "swoop-check")
	if err != nil {
		return e.Name, []finding{{ruleProgram, err.Error()}}, nil
	}
	defer os.RemoveAll(tmp)
	c := &checker{e: e, limit: limit, act: act, seen: map[string]bool{}}
	c.env = append(os.Environ(),
		ext.EnvExtDir+"="+e.Dir,
		"SWOOP_LAND="+filepath.Join(tmp, "land"),
	)
	// fzf gives a preview the size of its window. There is no window here.
	for _, size := range []string{"FZF_PREVIEW_COLUMNS=80", "FZF_PREVIEW_LINES=24"} {
		if key, _, _ := strings.Cut(size, "="); os.Getenv(key) == "" {
			c.env = append(c.env, size)
		}
	}

	c.files()
	items, ok := c.rows(ruleList, "list")
	c.rows(ruleList, "list", probe)
	if ok {
		c.icons(items)
		c.look(items)
	}
	for opened := 0; len(c.queue) > 0 && opened < maxViews; opened++ {
		id := c.queue[0]
		c.queue = c.queue[1:]
		c.view(id)
	}
	c.run()
	return e.Name, c.found, c.untried
}

// find is the extension at path, found the way the launcher finds one:
// a folder that holds a program of its own name, which can be run.
func find(path string) (ext.Extension, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ext.Extension{Name: filepath.Base(path)}, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return ext.Extension{Name: filepath.Base(abs)}, fmt.Errorf("%s is not there", path)
	}
	if !st.IsDir() {
		// The program was named. Its folder is the extension.
		abs = filepath.Dir(abs)
	}
	name := filepath.Base(abs)
	for _, e := range ext.DiscoverAll([]string{filepath.Dir(abs)}) {
		if e.Name == name {
			return e, nil
		}
	}
	return ext.Extension{Name: name}, fmt.Errorf("the folder %s holds no program called %s that can be run", name, name)
}

// answer is one run of the program.
type answer struct {
	out    []byte
	stderr string
	err    error // nil for exit 0
	late   bool  // still running at the limit
	held   bool  // gone, but something it started kept its output open
}

// ask runs the program with args and reads what it prints, as the
// launcher reads a list: until the output closes. A worker the program
// leaves behind with that output still open keeps the launcher waiting,
// so it is told apart here, after a second, and not waited for.
func (c *checker) ask(limit time.Duration, args ...string) answer {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.e.Exe, args...)
	cmd.Dir = c.e.Dir
	cmd.Env = c.env
	cmd.WaitDelay = time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	a := answer{out: out, err: err}
	a.stderr, _, _ = strings.Cut(strings.TrimSpace(stderr.String()), "\n")
	a.late = errors.Is(ctx.Err(), context.DeadlineExceeded)
	a.held = !a.late && errors.Is(err, exec.ErrWaitDelay)
	return a
}

// do runs the program with args and reads nothing: run and send, whose
// output is the terminal's and not the launcher's. A worker left behind
// holds nothing the launcher waits on.
func (c *checker) do(limit time.Duration, args ...string) answer {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.e.Exe, args...)
	cmd.Dir = c.e.Dir
	cmd.Env = c.env
	err := cmd.Run()
	return answer{err: err, late: errors.Is(ctx.Err(), context.DeadlineExceeded)}
}

// why says how a run failed, in a few words: the exit status, and the
// first line the program wrote to stderr.
func (a answer) why() string {
	msg := "it did not run"
	var exit *exec.ExitError
	switch {
	case errors.As(a.err, &exit):
		msg = "exit status " + strconv.Itoa(exit.ExitCode())
	case a.err != nil:
		msg = a.err.Error()
	}
	if a.stderr != "" {
		msg += ": " + a.stderr
	}
	return msg
}

// short is an id fit for one line of a finding.
func short(id string) string {
	if r := []rune(id); len(r) > 60 {
		id = string(r[:60]) + "…"
	}
	return strconv.Quote(id)
}

// rows runs a verb that prints rows, list or view, and holds it to what
// the launcher asks of one: exit 0, inside the limit, nothing left
// holding the output, and every line a row. ok is false when the output
// cannot be trusted, as the launcher does not trust it: no rows then.
func (c *checker) rows(rule string, args ...string) (items []protocol.Item, ok bool) {
	what := strings.Join(args, " ")
	a := c.ask(c.limit, args...)
	switch {
	case a.late:
		c.fail(ruleTime, "%s took longer than %s", what, c.limit)
		return nil, false
	case a.held:
		c.fail(ruleTime, "%s left a process behind with its output still open; the launcher waits for that too", what)
	case a.err != nil:
		c.fail(rule, "%s: %s", what, a.why())
		return nil, false
	}
	items = c.parse(what, a.out)
	c.unique(what, items)
	c.groups(what, items)
	return items, true
}

// parse reads what a verb printed into rows, and reports every line the
// launcher would skip without a word.
func (c *checker) parse(what string, out []byte) []protocol.Item {
	lines := strings.Split(string(out), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var items []protocol.Item
	bad := 0
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		fields := strings.Split(line, "\t")
		wrong := ""
		switch {
		case len(line) > maxLine:
			wrong = "is longer than 1 MiB, and the launcher reads no row after it"
		case !utf8.ValidString(line):
			wrong = "is not UTF-8"
		case line == "":
			wrong = "is empty"
		case len(fields) > protocol.Fields:
			wrong = fmt.Sprintf("has %d fields, not %d; a tab inside an id or a title makes one more", len(fields), protocol.Fields)
		case len(fields) == 1:
			wrong = fmt.Sprintf("has no tab in it; a row is %d fields with tabs between", protocol.Fields)
		case len(fields) < protocol.Fields:
			wrong = fmt.Sprintf("has %d fields, not %d", len(fields), protocol.Fields)
		case fields[0] == "":
			wrong = "has an empty id"
		}
		if wrong == "" {
			if it, err := protocol.Parse(line); err == nil {
				items = append(items, it)
				continue
			}
			wrong = "is not a row"
		}
		if bad++; bad <= maxShown {
			c.fail(ruleLine, "%s: line %d %s", what, i+1, wrong)
		}
	}
	if bad > maxShown {
		c.fail(ruleLine, "%s: %d more lines are not rows", what, bad-maxShown)
	}
	return items
}

// unique reports an id two rows of one list share. The id is all the
// launcher hands back: two rows with one id are one row to preview and
// to run.
func (c *checker) unique(what string, items []protocol.Item) {
	first := map[string]int{}
	shown := 0
	for i, it := range items {
		n, dup := first[it.ID]
		if !dup {
			first[it.ID] = i + 1
			continue
		}
		if shown++; shown <= maxShown {
			c.fail(ruleUnique, "%s: rows %d and %d have the same id, %s", what, n, i+1, short(it.ID))
		}
	}
}

// groups reports a header with no row under it: the last row, or one
// with another header next. The cursor never rests on a header, so it
// needs a row to go to.
func (c *checker) groups(what string, items []protocol.Item) {
	for i, it := range items {
		if it.Kind != "group" {
			continue
		}
		if i+1 == len(items) || items[i+1].Kind == "group" {
			c.fail(ruleGroup, "%s: the header %s has no row under it", what, short(it.Title))
		}
	}
}

// look takes a few rows of one list, the first of each kind, and asks
// for the preview and the actions of each, as a cursor on it would. A
// view row is queued to be opened.
func (c *checker) look(items []protocol.Item) {
	kinds := map[string]bool{}
	for _, it := range items {
		if it.Kind == "view" {
			c.enqueue(it.ID)
		}
		if it.Kind == "group" || kinds[it.Kind] || len(kinds) == 5 {
			continue
		}
		kinds[it.Kind] = true
		c.preview(it.ID)
		c.actions(it.ID)
		// A view opens, and a terminal row wants the terminal, which the
		// checker has not got. Any other row is one -act can run.
		if c.runnable == "" && it.Kind != "view" && it.Kind != "terminal" {
			c.runnable = it.ID
		}
	}
}

func (c *checker) enqueue(view string) {
	if !c.seen[view] {
		c.seen[view] = true
		c.queue = append(c.queue, view)
	}
}

// preview asks for the preview of an id the program printed. There is
// no limit on a preview in the contract, only the guard.
func (c *checker) preview(id string) {
	a := c.ask(guard, "preview", id)
	switch {
	case a.late:
		c.fail(rulePreview, "preview %s was not done in %s", short(id), guard)
	case a.held:
		// What a preview leaves running is its own business: fzf shows
		// what is there and reads on.
	case a.err != nil:
		c.fail(rulePreview, "preview %s: %s", short(id), a.why())
	}
}

// actions asks for the menu of an id the program printed. An exit other
// than 0, or nothing printed, is an extension with no menu for that row,
// which the contract allows. What is printed is rows, of three kinds.
func (c *checker) actions(id string) {
	what := "actions " + short(id)
	a := c.ask(c.limit, "actions", id)
	switch {
	case a.late:
		c.fail(ruleTime, "%s took longer than %s", what, c.limit)
		return
	case a.held:
		c.fail(ruleTime, "%s left a process behind with its output still open; the launcher waits for that too", what)
	case a.err != nil:
		return
	}
	for _, it := range c.parse(what, a.out) {
		switch it.Kind {
		case "action", "refresh", "terminal":
		default:
			c.fail(ruleActions, "%s: the row %s is of kind %q; an action is of kind action, refresh or terminal", what, short(it.ID), it.Kind)
		}
	}
}

// view opens one view: its rows with nothing typed and with a text, a
// few previews and menus, and, when its bar is a prompt, a send.
func (c *checker) view(id string) {
	pane := c.e.Pane(id)
	items, ok := c.rows(ruleView, "view", id)
	if !pane.Prompt {
		// A view whose bar is a prompt is never asked with the text.
		c.rows(ruleView, "view", id, probe)
	}
	if !ok {
		return
	}
	c.look(items)
	if !pane.Prompt {
		return
	}
	target := ""
	for _, it := range items {
		if it.Kind != "group" {
			target = it.ID
			break
		}
	}
	if target == "" {
		c.fail(ruleSend, "view %s has bar=prompt and prints no row; Enter sends the text to the row under the cursor, and there is none", id)
		return
	}
	if !c.act {
		c.leave("send")
		return
	}
	switch a := c.do(c.limit, "send", target, "swoop-check"); {
	case a.late:
		c.fail(ruleTime, "send %s took longer than %s; an answer that takes time is a worker's", short(target), c.limit)
	case a.err != nil:
		c.fail(ruleSend, "view %s has bar=prompt, and send %s: %s", id, short(target), a.why())
	}
}

// leave notes a verb that was not tried, once.
func (c *checker) leave(verb string) {
	for _, v := range c.untried {
		if v == verb {
			return
		}
	}
	c.untried = append(c.untried, verb)
}

// run does the first row found that Enter would run, with -act.
func (c *checker) run() {
	if c.runnable == "" {
		return
	}
	if !c.act {
		// run first: it reads better ahead of send.
		c.untried = append([]string{"run"}, c.untried...)
		return
	}
	switch a := c.do(guard, "run", c.runnable); {
	case a.late:
		c.fail(ruleRun, "run %s was not done in %s", short(c.runnable), guard)
	case a.err != nil:
		c.fail(ruleRun, "run %s: %s", short(c.runnable), a.why())
	}
}

// files reads the files beside the program, each the way the launcher
// reads it, and reports what the launcher would skip or misread without
// a word. A view named in one is queued to be opened.
func (c *checker) files() {
	if data, ok := c.read(ruleKeyword, ext.KeywordFile); ok {
		line, _, _ := strings.Cut(data, "\n")
		switch words := strings.Fields(line); {
		case len(words) == 0:
			c.fail(ruleKeyword, "the first line is empty; it holds the one word")
		case len(words) > 1:
			c.fail(ruleKeyword, "the first line holds %d words; the keyword is one, and the launcher takes %q", len(words), words[0])
		}
	}

	if data, ok := c.read(ruleKey, ext.KeyFile); ok {
		claimed := map[string]bool{}
		for i, line := range strings.Split(data, "\n") {
			f := strings.Fields(line)
			if len(f) == 0 || strings.HasPrefix(f[0], "#") {
				continue
			}
			switch {
			case len(f) < 2:
				c.fail(ruleKey, "line %d names no view; a line is <key> <view-id> <title>", i+1)
				continue
			case !ext.ValidKey(f[0]):
				c.fail(ruleKey, "line %d: %s is not a key an extension may claim", i+1, f[0])
			case claimed[f[0]]:
				c.fail(ruleKey, "line %d claims %s again; the first line has it", i+1, f[0])
			}
			claimed[f[0]] = true
			c.enqueue(f[1])
		}
	}

	if data, ok := c.read(ruleViews, ext.ViewsFile); ok {
		named := map[string]bool{}
		for i, line := range strings.Split(data, "\n") {
			f := strings.Fields(line)
			if len(f) == 0 || strings.HasPrefix(f[0], "#") {
				continue
			}
			if named[f[0]] {
				c.fail(ruleViews, "line %d is about %s again; the launcher reads the first line", i+1, f[0])
			}
			named[f[0]] = true
			for _, s := range f[1:] {
				if wrong := setting(s); wrong != "" {
					c.fail(ruleViews, "line %d: %s", i+1, wrong)
				}
			}
			c.enqueue(f[0])
		}
	}

	if data, ok := c.read(ruleCache, ext.CacheFile); ok && !c.e.Once() {
		c.fail(ruleCache, "it holds %q; the word is run", first(data))
	}

	if data, ok := c.read(ruleIcons, ext.IconsFile); ok {
		switch {
		case first(data) != "id":
			c.fail(ruleIcons, "it holds %q; the word is id", first(data))
		case !c.e.Once():
			c.fail(ruleIcons, "it needs a cache file beside it that holds run: the terminal takes the pictures once, as the launcher starts")
		}
	}
}

// read is the text of a file beside the program. ok is false when there
// is none, which is the common case and breaks nothing.
func (c *checker) read(rule, name string) (text string, ok bool) {
	data, err := os.ReadFile(filepath.Join(c.e.Dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		c.fail(rule, "%v", err)
		return "", false
	}
	return string(data), true
}

// first is the first word of the first line, as the launcher reads a
// file that holds one word.
func first(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	if words := strings.Fields(line); len(words) > 0 {
		return words[0]
	}
	return ""
}

// setting says what is wrong with one setting of a views line, "" for
// nothing. The launcher skips a setting it does not know, so a file
// written for a newer contract still reads; here it is reported, since
// to this contract it is a typing mistake that changes nothing.
func setting(s string) string {
	key, value, _ := strings.Cut(s, "=")
	switch key {
	case "bar":
		if value != "prompt" {
			return fmt.Sprintf("%s is not a setting of contract %s; a bar is bar=prompt or left out", s, contract)
		}
	case "preview":
		for _, v := range strings.Split(value, ",") {
			n, ok := strings.CutSuffix(v, "%")
			switch {
			case v == "wrap", v == "follow":
			case ok:
				if pct, err := strconv.Atoi(n); err != nil || pct < settings.PreviewMin || pct > settings.PreviewMax {
					return fmt.Sprintf("preview=%s: a width is %d%% to %d%%", v, settings.PreviewMin, settings.PreviewMax)
				}
			default:
				return fmt.Sprintf("preview=%s is not a setting of contract %s; a preview is a width in %%, wrap, follow", v, contract)
			}
		}
	default:
		return fmt.Sprintf("%s is not a setting of contract %s", s, contract)
	}
	return ""
}

// icons holds the rows of an extension that asked for pictures to what
// the icons file says of them: each id is the path of a file.
func (c *checker) icons(items []protocol.Item) {
	if !c.e.IconsByID() {
		return
	}
	for _, it := range items {
		path := it.ID
		if !filepath.IsAbs(path) {
			path = filepath.Join(c.e.Dir, path)
		}
		if _, err := os.Stat(path); err != nil {
			c.fail(ruleIcons, "the row %s has an id that is not the path of a file, and the icons file says each is", short(it.ID))
			return
		}
	}
}
