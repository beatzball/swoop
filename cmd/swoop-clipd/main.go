// swoop-clipd keeps the clipboard history. It is the first part of swoop
// that runs while the launcher is closed: a watcher that notices when the
// clipboard changes and appends the new text to the history file. The
// extension reads the file back through the same program.
//
//	swoop-clipd start           start the watcher detached, if none runs
//	swoop-clipd run             the watcher itself, in the foreground
//	swoop-clipd list [query]    result lines, newest first
//	swoop-clipd get <id>        the full text of one entry
//	swoop-clipd status          say whether a watcher runs
//
// Privacy: an entry whose pasteboard types include the concealed or
// transient marks that password managers set is never stored. Entries are
// cut at 64 KB and the file is kept to 500 entries. The file is the user's
// own, mode 0600, under the data directory.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/beatzball/swoop/internal/clip"
	"github.com/beatzball/swoop/internal/protocol"
)

// interval is how often the watcher looks at the pasteboard. Asking for
// the change count is a cheap call; a third of a second is quick enough
// that a copy is in the history before anyone could open the launcher.
const interval = 300 * time.Millisecond

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: swoop-clipd start|run|list [query]|get <id>|status")
		os.Exit(2)
	}
	store := clip.Default()
	var err error
	switch os.Args[1] {
	case "start":
		err = start(store)
	case "run":
		err = run(store)
	case "status":
		if running(store) {
			fmt.Println("running")
		} else {
			fmt.Println("not running")
			os.Exit(1)
		}
	case "list":
		q := ""
		if len(os.Args) > 2 {
			q = os.Args[2]
		}
		err = list(store, q)
	case "get":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: swoop-clipd get <id>")
			os.Exit(2)
		}
		var e clip.Entry
		if e, err = store.Get(os.Args[2]); err == nil {
			fmt.Print(e.Text)
		}
	default:
		fmt.Fprintln(os.Stderr, "swoop-clipd: unknown command", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-clipd:", err)
		os.Exit(1)
	}
}

// run is the watcher: whenever the pasteboard's change count moves and
// the content is plain text that is not marked private, store it.
func run(store clip.Store) error {
	unlock, err := lock(store)
	if err != nil {
		return err
	}
	defer unlock()
	pb, err := openPasteboard()
	if err != nil {
		return err
	}
	last := pb.ChangeCount()
	for {
		time.Sleep(interval)
		n := pb.ChangeCount()
		if n == last {
			continue
		}
		last = n
		if pb.Concealed() {
			continue
		}
		text, ok := pb.Text()
		if !ok {
			continue
		}
		if err := store.Append(text, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, "swoop-clipd:", err)
		}
	}
}

// list prints the history as result lines. The id is the entry's time; the
// title is its first line; the subtitle is when it was copied.
func list(store clip.Store, query string) error {
	entries, err := store.Find(query)
	if err != nil {
		return err
	}
	items := make([]protocol.Item, 0, len(entries))
	for _, e := range entries {
		items = append(items, protocol.Item{
			ID:       strconv.FormatInt(e.ID, 10),
			Kind:     "text",
			Icon:     "", // nf-fa-clipboard
			Title:    clip.Title(e.Text, 60),
			Subtitle: when(time.Unix(0, e.ID)),
		})
	}
	return protocol.Write(os.Stdout, items)
}

// when says how long ago, in the words a person would use.
func when(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return t.Format("15:04")
	default:
		return t.Format("Jan 2")
	}
}
