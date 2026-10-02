// swoop-ai is the Ask AI pane: the extension behind Tab. It speaks the
// extension contract, so the launcher routes to it like any other, and
// the launcher knows nothing of it. Three files beside the shim in
// extensions/ai say what is its own: `key` claims Tab for the ask view,
// `views` says that view's bar is a prompt and its preview is wrapped
// and follows, and `keyword` gives it "ai". `send` is the contract's
// verb for a pane whose bar is a prompt: Enter there.
//
//	swoop-ai list                       nothing: Tab is the way in
//	swoop-ai view ask [text]            rows: New conversation, then the past ones
//	swoop-ai preview <id>               the transcript; dots while the answer is on its way
//	swoop-ai actions <id>               Copy last answer, Copy conversation, Delete
//	swoop-ai run <id> [action]          copy, copy-all, delete
//	swoop-ai send <id|new> <prompt>     record the prompt and start the worker; returns at once
//	swoop-ai work <id>                  the worker: run the model, stream the answer into the file
//
// swoop does not know what a model is. The worker runs one command, the
// line in ~/.config/swoop/ai, with the conversation on its stdin, and puts
// what comes out of its stdout into the file as it comes. See command.go
// for the defaults when there is no such line.
//
// The pane redraws through fzf's own HTTP API: bin/swoop starts fzf with
// --listen, fzf hands the socket and key to its children, and the worker
// posts refresh-preview as the answer arrives. That is what animates the
// dots and streams the text without the preview command ever blocking.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/chat"
	"github.com/beatzball/swoop/internal/markdown"
	"github.com/beatzball/swoop/internal/models"
	"github.com/beatzball/swoop/internal/paste"
	"github.com/beatzball/swoop/internal/protocol"
	"github.com/beatzball/swoop/internal/settings"
	"github.com/beatzball/swoop/internal/usage"
)

// newID is the row that starts a conversation.
const newID = "new"

// askView is the pane's view, as the launcher names it: what an open is
// counted under in the usage log.
const askView = "ext/ai/ask"

// envLand names the file a send writes the conversation's id to, so the
// launcher puts the cursor on it once the list is back: a new
// conversation is a new row, and an old one moves to the top.
const envLand = "SWOOP_LAND"

func main() {
	if len(os.Args) < 2 {
		usageExit()
	}
	store := chat.Store{Dir: chat.DefaultDir()}
	if store.Dir == "" {
		fatal("no home directory")
	}
	arg := func(i int) string {
		if len(os.Args) > i {
			return os.Args[i]
		}
		return ""
	}
	var err error
	switch os.Args[1] {
	case "list":
	case "view":
		err = view(store)
	case "preview":
		err = preview(store, arg(2))
	case "actions":
		err = actions(arg(2))
	case "run":
		err = run(store, arg(2), arg(3))
	case "send":
		err = send(store, arg(2), strings.TrimSpace(arg(3)))
	case "work":
		err = work(store, arg(2))
	default:
		usageExit()
	}
	if err != nil {
		fatal(err.Error())
	}
}

func usageExit() {
	fmt.Fprintln(os.Stderr, "usage: swoop-ai list | view ask [text] | preview <id> | actions <id> | run <id> [action] | send <id|new> <prompt>")
	os.Exit(2)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "swoop-ai:", msg)
	os.Exit(1)
}

// view prints the pane's rows. The text in the bar is a prompt, not a
// filter, so it is ignored: every conversation is always listed.
func view(store chat.Store) error {
	items := []protocol.Item{{ID: newID, Kind: "new", Icon: "", Title: "New conversation", Subtitle: "type, then Enter · " + shortModel()}}
	all, err := store.List()
	if err != nil {
		return err
	}
	for _, c := range all {
		items = append(items, protocol.Item{ID: c.ID, Kind: "conversation", Icon: "󰭹", Title: c.Title, Subtitle: subtitle(c)})
	}
	return protocol.Write(os.Stdout, items)
}

func subtitle(c *chat.Conversation) string {
	if c.Pending {
		return "thinking…"
	}
	turns := len(c.Turns) / 2
	s := fmt.Sprintf("%d turn", turns)
	if turns != 1 {
		s += "s"
	}
	return s + " · " + ago(c.Updated)
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// shortModel is the current model for the New row's subtitle: the
// setting, or "default" when there is none.
func shortModel() string {
	if v := settings.Get(settings.AI, ""); v != "" {
		return v
	}
	return "default model"
}

// dots are the frames under a prompt while the model has said nothing
// yet. The worker's tick picks the frame.
var dots = []string{"·", "··", "···"}

// waitingLine is what shows under a prompt with no answer yet: the dots,
// how long it has been, and the last thing the command said on stderr,
// which for the default command is the tool it is using.
func waitingLine(c *chat.Conversation) string {
	// Padded to three cells, so the time after the dots does not move as
	// they grow and shrink.
	line := fmt.Sprintf("%-3s", dots[c.Tick%len(dots)])
	if !c.Asked.IsZero() {
		line += " " + elapsed(time.Since(c.Asked))
	}
	if c.Status != "" {
		line += " · " + c.Status
	}
	return line
}

func elapsed(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}

// preview prints the transcript: each prompt in bold, each answer as
// markdown rendered at the pane's width, the dots while an answer has
// not started. The preview window follows, so a growing
// answer keeps its end in view.
func preview(store chat.Store, id string) error {
	if id == newID || id == "" {
		fmt.Printf("Type a question and press Enter.\n\n\x1b[2mModel: %s\nctrl-k changes it.\x1b[22m\n", models.Current())
		return nil
	}
	c, err := store.Load(id)
	if err != nil {
		return err
	}
	render := renderer()
	for _, t := range c.Turns {
		if t.Role == "user" {
			fmt.Printf("\x1b[1m%s\x1b[22m\n\n", strings.TrimSpace(t.Text))
			continue
		}
		if t.Text == "" && c.Pending {
			if c.Worker != 0 && !alive(c.Worker) {
				fmt.Print("\x1b[31mThe worker stopped before an answer came. Ask again.\x1b[39m\n\n")
				continue
			}
			fmt.Printf("\x1b[2m%s\x1b[22m\n\n", waitingLine(c))
			continue
		}
		fmt.Print(render(strings.TrimSpace(t.Text)))
	}
	if c.Error != "" {
		fmt.Printf("\x1b[31m%s\x1b[39m\n", c.Error)
	}
	return nil
}

// renderer turns markdown into styled text for the pane, at the width
// fzf gives the preview. With `render = <command>` in the settings file
// the command does it: the answer on its stdin, its stdout shown, so
// glow, swoop-md, or anything else can draw the transcript. Without it
// the built-in renderer runs here, no process, which is what a redraw
// every hundred milliseconds wants. A command that fails leaves the
// answer as it is, plain.
func renderer() func(string) string {
	cols := 60
	if v, err := strconv.Atoi(os.Getenv("FZF_PREVIEW_COLUMNS")); err == nil && v > 10 {
		cols = v
	}
	if line := settings.Get(settings.Render, ""); line != "" {
		return func(s string) string {
			cmd := exec.Command("sh", "-c", line)
			cmd.Stdin = strings.NewReader(s)
			cmd.Env = append(os.Environ(), "FZF_PREVIEW_COLUMNS="+strconv.Itoa(cols), "COLUMNS="+strconv.Itoa(cols))
			out, err := cmd.Output()
			if err != nil {
				return s + "\n\n"
			}
			return strings.TrimRight(string(out), "\n") + "\n\n"
		}
	}
	return func(s string) string {
		return markdown.Render(s, cols) + "\n"
	}
}

// actions is the ctrl-k menu: for New conversation, the models to pick
// from, each a refresh row so Enter sets it and returns to the pane; for
// a conversation, copy and delete.
func actions(id string) error {
	if id == "" {
		return nil
	}
	if id == newID {
		current := settings.Get(settings.AI, "")
		var items []protocol.Item
		for _, m := range models.Choices("") {
			note := m.Note
			if m.Value == current {
				note = strings.TrimSpace("current  " + note)
			}
			items = append(items, protocol.Item{ID: "model:" + m.Value, Kind: "refresh", Icon: "󰭹", Title: m.Title, Subtitle: note})
		}
		return protocol.Write(os.Stdout, items)
	}
	return protocol.Write(os.Stdout, conversationActions(paste.Missing()))
}

// conversationActions is the ctrl-k menu on a conversation. missing is
// why this system cannot copy, from paste.Missing; when it says
// something, both copy rows say it, so the reason is on screen before
// Enter rather than lost on a stderr nobody sees.
func conversationActions(missing string) []protocol.Item {
	copyNote, copyAllNote := "Enter", ""
	if missing != "" {
		copyNote, copyAllNote = missing, missing
	}
	return []protocol.Item{
		{ID: "copy", Kind: "action", Icon: "", Title: "Copy last answer", Subtitle: copyNote},
		{ID: "copy-all", Kind: "action", Icon: "", Title: "Copy conversation", Subtitle: copyAllNote},
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: ""},
	}
}

func run(store chat.Store, id, action string) error {
	if id == "" {
		return nil
	}
	if id == newID {
		// The one thing New runs: a model picked from its ctrl-k menu.
		if value, ok := strings.CutPrefix(action, "model:"); ok {
			return settings.Set(settings.AI, value)
		}
		return nil
	}
	c, err := store.Load(id)
	if err != nil {
		return err
	}
	switch action {
	case "", "copy":
		return paste.Copy(c.Answer())
	case "copy-all":
		return paste.Copy(c.Transcript())
	case "delete":
		return store.Delete(id)
	}
	return fmt.Errorf("unknown action %q", action)
}

// send records the prompt on the conversation, or on a new one, and
// starts the worker detached, so the launcher's Enter returns at once.
func send(store chat.Store, id, prompt string) error {
	if prompt == "" {
		return nil
	}
	var c *chat.Conversation
	var err error
	if id == newID || id == "" {
		c, err = store.New(prompt)
	} else {
		c, err = store.Load(id)
	}
	if err != nil {
		return err
	}
	if c.Pending {
		// One answer at a time per conversation; a prompt sent while the
		// model is still writing is dropped, and the bar keeps it.
		return fmt.Errorf("still answering")
	}
	c.Ask(prompt)
	_ = usage.Record(askView, "ai", "Ask AI")
	if err := store.Save(c); err != nil {
		return err
	}
	if file := os.Getenv(envLand); file != "" {
		// Not worth failing the send for: without it the cursor stays on
		// its row number.
		_ = os.WriteFile(file, []byte(c.ID+"\n"), 0o600)
	}
	return startWorker(c.ID)
}
