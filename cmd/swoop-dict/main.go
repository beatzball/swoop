// swoop-dict prints the definition of a word from the system dictionary,
// wrapped to the width of the fzf preview pane, and exits. On macOS the
// words come from the same Dictionary app the user already has; there is
// nothing to download. Other OSes get a stub until someone adds a source.
//
//	swoop-dict <word>
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: swoop-dict <word>")
		os.Exit(2)
	}
	text, err := define(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "swoop-dict:", err)
		os.Exit(1)
	}
	width := 60
	if v, err := strconv.Atoi(os.Getenv("FZF_PREVIEW_COLUMNS")); err == nil && v > 10 {
		width = v - 1
	}
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	for _, para := range split(text) {
		fmt.Fprintln(out, wrap(para, width))
	}
}

// split breaks the dictionary's one long line into paragraphs at the
// sense markers it uses: numbered senses and bullets. The headword and
// its pronunciation stay on the first line.
func split(text string) []string {
	text = strings.TrimSpace(text)
	var paras []string
	for _, marker := range []string{" • ", " 1 ", " 2 ", " 3 ", " 4 ", " 5 ", " 6 ", " 7 ", " 8 ", " 9 "} {
		text = strings.ReplaceAll(text, marker, "\n"+strings.TrimLeft(marker, " "))
	}
	for _, p := range strings.Split(text, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			paras = append(paras, p)
		}
	}
	return paras
}

// wrap folds s at spaces so no line is longer than width.
func wrap(s string, width int) string {
	var b strings.Builder
	line := 0
	for _, word := range strings.Fields(s) {
		n := len([]rune(word))
		if line > 0 && line+1+n > width {
			b.WriteByte('\n')
			line = 0
		} else if line > 0 {
			b.WriteByte(' ')
			line++
		}
		b.WriteString(word)
		line += n
	}
	return b.String()
}
