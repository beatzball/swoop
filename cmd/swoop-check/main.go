// swoop-check is the conformance test for the extension contract, the
// file CONTRACT.md at the repository root. It runs an extension the way
// the launcher does and prints where the extension does not meet the
// contract, one finding per line with the rule it breaks:
//
//	swoop-check [-act] [-limit 2s] <extension>...
//
//	hello: line: list: line 2 has 4 fields, not 5
//
// An extension is named by its folder, or by the program in it. Exit 0
// when every one meets the contract, 1 when one does not, 2 for a call
// that makes no sense.
//
// It asks only the verbs that read: list, view, preview, actions. run
// and send do things, and the checker cannot know what: Sleep puts the
// machine to sleep. -act tries them too, for real, on the first row it
// finds; it is for an extension whose run is safe to do once.
//
// -limit is the time list, view, actions and send get. The contract's is
// two seconds, the default; a test on a busy machine gives more, so a
// slow runner is not taken for a slow extension.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	act := flag.Bool("act", false, "also try run and send, for real, on the first row found")
	limit := flag.Duration("limit", contractLimit, "the time list, view, actions and send get")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: swoop-check [-act] [-limit 2s] <extension>...")
		fmt.Fprintln(os.Stderr, "checks each extension, a folder or the program in it, against contract "+contract)
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 || *limit <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	if failed := report(os.Stdout, flag.Args(), *limit, *act); failed > 0 {
		fmt.Fprintf(os.Stderr, "swoop-check: %d of %d do not meet contract %s\n", failed, flag.NArg(), contract)
		os.Exit(1)
	}
}

// report checks each extension and writes what it found to w: a line per
// finding, or one line that says the extension meets the contract. It
// returns how many do not.
func report(w io.Writer, paths []string, limit time.Duration, act bool) (failed int) {
	for _, path := range paths {
		name, found, untried := check(path, limit, act)
		for _, f := range found {
			fmt.Fprintf(w, "%s: %s: %s\n", name, f.rule, f.text)
		}
		if len(found) > 0 {
			failed++
			continue
		}
		note := ""
		if len(untried) > 0 {
			note = " (not tried without -act: " + strings.Join(untried, ", ") + ")"
		}
		fmt.Fprintf(w, "%s: meets contract %s%s\n", name, contract, note)
	}
	return failed
}

// contractLimit is the contract's: what the launcher gives list, view,
// actions and send. See internal/ext.
const contractLimit = 2 * time.Second
