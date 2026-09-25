// swoop-clipd keeps the clipboard history. It is the first part of swoop
// that runs while the launcher is closed: a watcher that notices when the
// clipboard changes and appends the new text to the history file. The
// extension reads the file back through the same program.
//
//	swoop-clipd start           start the watcher detached, if none runs
//	swoop-clipd run             the watcher itself, in the foreground
//	swoop-clipd list [query]    result lines, newest first
//	swoop-clipd get <id>        the full text of one entry
//	swoop-clipd delete <id>     remove one entry
//	swoop-clipd clear           remove every entry
//	swoop-clipd status          say whether a watcher runs
//	swoop-clipd types           the marks on the clipboard right now
//	swoop-clipd frontmost       the bundle id of the app in front
//
// Privacy, two rules. An entry whose pasteboard types include the concealed
// or transient marks (org.nspasteboard.ConcealedType, TransientType) is
// never stored; some password managers set them. And a copy made while an
// app on the ignore list is in front is never stored; the list defaults to
// the known password managers and lives in ~/.config/swoop/clipboard.ignore,
// because not every manager sets the mark. Entries are cut at 64 KB and the
// file is kept to 500 entries, mode 0600, under the data directory.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
		fmt.Fprintln(os.Stderr, "usage: swoop-clipd start|run|list [query]|get <id>|delete <id>|clear|status|types|frontmost")
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
	case "delete":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: swoop-clipd delete <id>")
			os.Exit(2)
		}
		err = store.Delete(os.Args[2])
	case "clear":
		err = store.Clear()
	case "types":
		var pb pasteboard
		if pb, err = openPasteboard(); err == nil {
			fmt.Println(pb.Types())
		}
	case "frontmost":
		var pb pasteboard
		if pb, err = openPasteboard(); err == nil {
			fmt.Println(pb.Frontmost())
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
// the content is plain text that is not marked private and was not copied
// while an ignored app was in front, store it.
//
// "In front" is checked twice: the app in front now, and the one in front
// at the previous tick. A password manager that copies and then hides its
// window has already handed the front to another app by the time the
// change is noticed, and the previous tick still remembers it.
//
// The ignore list is read on every change, so an edit to it takes effect
// without a restart; it is a few lines and a change is rare. With
// SWOOP_CLIPD_DEBUG set, every change is logged with the apps and the
// marks seen, never the text, which is how a new manager is diagnosed.
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
	debug := os.Getenv("SWOOP_CLIPD_DEBUG") != ""
	last := pb.ChangeCount()
	prevFront := pb.Frontmost()
	for {
		time.Sleep(interval)
		front := pb.Frontmost()
		n := pb.ChangeCount()
		if n == last {
			prevFront = front
			continue
		}
		last = n
		concealed := pb.Concealed()
		ignore, err := clip.LoadIgnore(clip.IgnorePath())
		if err != nil {
			fmt.Fprintln(os.Stderr, "swoop-clipd:", err)
		}
		ignored := ignore[strings.ToLower(front)] || ignore[strings.ToLower(prevFront)]
		if debug {
			fmt.Fprintf(os.Stderr, "%s change=%d front=%q prev=%q concealed=%v ignored=%v types=%q\n",
				time.Now().Format("15:04:05.000"), n, front, prevFront, concealed, ignored,
				strings.ReplaceAll(pb.Types(), "\n", " "))
		}
		prevFront = front
		if concealed || ignored {
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
