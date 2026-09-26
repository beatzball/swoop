// swoop-ai is the Ask AI pane: the extension behind Tab. It speaks the
// extension contract, so the launcher routes to it like any other, and
// one verb more, `send`, which the pane's Enter uses.
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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/chat"
	"github.com/beatzball/swoop/internal/markdown"
	"github.com/beatzball/swoop/internal/protocol"
)

// newID is the row that starts a conversation.
const newID = "new"

func main() {
	if len(os.Args) < 2 {
		usage()
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
		usage()
	}
	if err != nil {
		fatal(err.Error())
	}
}

func usage() {
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
	items := []protocol.Item{{ID: newID, Kind: "new", Icon: "", Title: "New conversation", Subtitle: "type, then Enter"}}
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
		fmt.Println("Type a question and press Enter.")
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
// fzf gives the preview.
func renderer() func(string) string {
	cols := 60
	if v, err := strconv.Atoi(os.Getenv("FZF_PREVIEW_COLUMNS")); err == nil && v > 10 {
		cols = v
	}
	return func(s string) string {
		return markdown.Render(s, cols) + "\n"
	}
}

// actions is the ctrl-k menu for a conversation. The New row has none.
func actions(id string) error {
	if id == newID || id == "" {
		return nil
	}
	return protocol.Write(os.Stdout, []protocol.Item{
		{ID: "copy", Kind: "action", Icon: "", Title: "Copy last answer", Subtitle: "Enter"},
		{ID: "copy-all", Kind: "action", Icon: "", Title: "Copy conversation", Subtitle: ""},
		{ID: "delete", Kind: "refresh", Icon: "", Title: "Delete", Subtitle: ""},
	})
}

func run(store chat.Store, id, action string) error {
	if id == newID || id == "" {
		return nil
	}
	c, err := store.Load(id)
	if err != nil {
		return err
	}
	switch action {
	case "", "copy":
		return copyText(c.Answer())
	case "copy-all":
		return copyText(c.Transcript())
	case "delete":
		return store.Delete(id)
	}
	return fmt.Errorf("unknown action %q", action)
}

// copyText puts text on the clipboard with whatever this system has.
func copyText(text string) error {
	var cmd *exec.Cmd
	switch {
	case runtime.GOOS == "darwin":
		cmd = exec.Command("pbcopy")
	case runtime.GOOS == "windows":
		cmd = exec.Command("clip")
	case exec.Command("wl-copy").Err == nil && os.Getenv("WAYLAND_DISPLAY") != "":
		cmd = exec.Command("wl-copy")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
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
	if err := store.Save(c); err != nil {
		return err
	}
	return startWorker(c.ID)
}
