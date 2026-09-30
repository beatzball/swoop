// swoop-match filters rows for a view, the way the Go views filter their
// own: fzf's matching is off inside a view, so the view keeps the rows
// that answer to the text in the bar. It is for an extension written in
// shell, which prints every row and pipes them through this.
//
//	swoop-match <query>   stdin's protocol lines, the matching ones out, best first
//
// A row matches when every word of the query matches its title or its
// subtitle, in any order and any case; a word matches when its letters
// are in the text in that order (see internal/match for the rules and
// the ranking). A row of kind "group" is a header: the rows under it are
// ranked under it, and a header with no matching row is left out. An
// empty query passes every row through.
//
// A row the view makes from the text typed (an Add row) is not for this
// filter: the view prints it itself, before or after the pipe.
//
// A line that is not a protocol line is dropped, as the launcher drops it.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/beatzball/swoop/internal/match"
	"github.com/beatzball/swoop/internal/protocol"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: swoop-match <query> < rows")
		os.Exit(2)
	}
	if err := run(strings.Join(os.Args[1:], " "), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "swoop-match:", err)
		os.Exit(1)
	}
}

func run(query string, r io.Reader, w io.Writer) error {
	var items []protocol.Item
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		if it, err := protocol.Parse(sc.Text()); err == nil {
			items = append(items, it)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return protocol.Write(w, match.Items(match.New(query), items))
}
