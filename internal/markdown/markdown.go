// Package markdown turns markdown into styled text for a terminal pane.
// goldmark parses; this file walks the tree and prints: headings bold,
// emphasis and strong, code spans in colour, code blocks indented in one
// colour with no highlighting, lists with nesting, block quotes with a
// bar, rules, links as text with the address dim after, tables with box
// lines, paragraphs wrapped to the width.
//
// It is here in place of glamour, which draws all of this beautifully and
// brings a code highlighter with a lexer for every language along: seven
// megabytes of binary for a transcript that needs its lists lined up.
package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// The styles. Escape sequences rather than a styling library: there are
// nine of them, and a pane redraw runs this on every refresh.
const (
	bold      = "\x1b[1m"
	unbold    = "\x1b[22m"
	dim       = "\x1b[2m"
	undim     = "\x1b[22m"
	italic    = "\x1b[3m"
	unitalic  = "\x1b[23m"
	underline = "\x1b[4m"
	nounder   = "\x1b[24m"
	codeFg    = "\x1b[38;5;215m" // a warm orange for code, on any background
	quoteFg   = "\x1b[38;5;110m" // a soft blue for quotes and rules
	linkFg    = "\x1b[38;5;75m"
	reset     = "\x1b[39m"
)

// Render prints md as styled text, lines no wider than width where the
// text allows it. Width below 20 is treated as 20.
func Render(md string, width int) string {
	if width < 20 {
		width = 20
	}
	src := []byte(md)
	parser := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough)).Parser()
	doc := parser.Parse(text.NewReader(src))
	r := &renderer{src: src, width: width}
	r.blocks(doc, "", "")
	return strings.TrimRight(r.out.String(), "\n") + "\n"
}

type renderer struct {
	src   []byte
	width int
	out   strings.Builder
}

// blocks prints the children of n, each block separated by a blank line.
// first is the prefix of a block's first line, rest of the others: how a
// list item's bullet and a quote's bar get in front of wrapped text.
func (r *renderer) blocks(n ast.Node, first, rest string) {
	i := 0
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if i > 0 {
			r.out.WriteString(strings.TrimRight(rest, " ") + "\n")
		}
		p1, p2 := first, rest
		if i > 0 {
			p1 = rest
		}
		r.block(c, p1, p2)
		i++
	}
}

func (r *renderer) block(n ast.Node, first, rest string) {
	switch n := n.(type) {
	case *ast.Heading:
		s := bold + r.inlines(n) + unbold
		if n.Level == 1 {
			s = bold + underline + r.inlines(n) + nounder + unbold
		}
		r.wrap(s, first, rest)
	case *ast.Paragraph, *ast.TextBlock:
		r.wrap(r.inlines(n), first, rest)
	case *ast.CodeBlock, *ast.FencedCodeBlock:
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			p := rest
			if i == 0 {
				p = first
			}
			r.out.WriteString(p + "  " + codeFg + strings.TrimRight(string(seg.Value(r.src)), "\n") + reset + "\n")
		}
	case *ast.Blockquote:
		bar := quoteFg + "│ " + reset
		r.blocks(n, first+bar, rest+bar)
	case *ast.List:
		r.list(n, first, rest)
	case *ast.ThematicBreak:
		w := r.width - ansi.StringWidth(first)
		if w < 4 {
			w = 4
		}
		r.out.WriteString(first + quoteFg + strings.Repeat("─", w) + reset + "\n")
	case *ast.HTMLBlock:
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			r.wrap(dim+strings.TrimRight(string(seg.Value(r.src)), "\n")+undim, first, rest)
		}
	case *east.Table:
		r.table(n, first, rest)
	default:
		r.wrap(r.inlines(n), first, rest)
	}
}

// list prints items with a bullet or a number, and nested blocks under
// them indented to the text.
func (r *renderer) list(n *ast.List, first, rest string) {
	num := n.Start
	if num == 0 {
		num = 1
	}
	i := 0
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		marker := "• "
		if n.IsOrdered() {
			marker = fmt.Sprintf("%d. ", num)
			num++
		}
		p := rest
		if i == 0 {
			p = first
		}
		indent := strings.Repeat(" ", ansi.StringWidth(marker))
		if n.IsTight {
			r.tightItem(item, p+marker, rest+indent)
		} else {
			if i > 0 {
				r.out.WriteString(strings.TrimRight(rest, " ") + "\n")
			}
			r.blocks(item, p+marker, rest+indent)
		}
		i++
	}
}

// tightItem prints an item of a tight list: no blank lines between the
// item's own blocks either.
func (r *renderer) tightItem(item ast.Node, first, rest string) {
	i := 0
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		p := rest
		if i == 0 {
			p = first
		}
		r.block(c, p, rest)
		i++
	}
}

// table draws a box: the header bold, a rule under it, columns padded to
// the widest cell. Cells wider than the pane are not wrapped; the pane
// scrolls sideways or cuts, and a table that wide is rare in an answer.
func (r *renderer) table(t *east.Table, first, rest string) {
	var rows [][]string
	for row := t.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, r.inlines(cell))
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return
	}
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	widths := make([]int, cols)
	for _, row := range rows {
		for i, c := range row {
			if w := ansi.StringWidth(c); w > widths[i] {
				widths[i] = w
			}
		}
	}
	line := func(cells []string, style string) string {
		var b strings.Builder
		for i := 0; i < cols; i++ {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			if i > 0 {
				b.WriteString(" " + quoteFg + "│" + reset + " ")
			}
			b.WriteString(style + c + strings.Repeat(" ", widths[i]-ansi.StringWidth(c)))
			if style != "" {
				b.WriteString(unbold)
			}
		}
		return b.String()
	}
	for i, row := range rows {
		p := rest
		if i == 0 {
			p = first
		}
		style := ""
		if i == 0 {
			style = bold
		}
		r.out.WriteString(p + line(row, style) + "\n")
		if i == 0 {
			var b strings.Builder
			for j, w := range widths {
				if j > 0 {
					b.WriteString("─" + "┼" + "─")
				}
				b.WriteString(strings.Repeat("─", w))
			}
			r.out.WriteString(rest + quoteFg + b.String() + reset + "\n")
		}
	}
}

// inlines prints the inline content of a block as one styled string.
func (r *renderer) inlines(n ast.Node) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		r.inline(c, &b)
	}
	return b.String()
}

func (r *renderer) inline(n ast.Node, b *strings.Builder) {
	switch n := n.(type) {
	case *ast.Text:
		b.Write(n.Segment.Value(r.src))
		if n.SoftLineBreak() {
			b.WriteByte(' ')
		}
		if n.HardLineBreak() {
			b.WriteByte('\n')
		}
	case *ast.String:
		b.Write(n.Value)
	case *ast.Emphasis:
		open, close := italic, unitalic
		if n.Level >= 2 {
			open, close = bold, unbold
		}
		b.WriteString(open)
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			r.inline(c, b)
		}
		b.WriteString(close)
	case *ast.CodeSpan:
		b.WriteString(codeFg)
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				b.Write(t.Segment.Value(r.src))
			}
		}
		b.WriteString(reset)
	case *ast.Link:
		var inner strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			r.inline(c, &inner)
		}
		label := inner.String()
		dest := string(n.Destination)
		if label == "" || label == dest {
			b.WriteString(linkFg + underline + dest + nounder + reset)
		} else {
			b.WriteString(linkFg + underline + label + nounder + reset + dim + " (" + dest + ")" + undim)
		}
	case *ast.AutoLink:
		b.WriteString(linkFg + underline + string(n.URL(r.src)) + nounder + reset)
	case *ast.Image:
		var inner strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			r.inline(c, &inner)
		}
		b.WriteString(dim + "[image: " + inner.String() + "]" + undim)
	case *ast.RawHTML:
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			b.WriteString(dim + string(seg.Value(r.src)) + undim)
		}
	case *east.Strikethrough:
		b.WriteString("\x1b[9m")
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			r.inline(c, b)
		}
		b.WriteString("\x1b[29m")
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			r.inline(c, b)
		}
	}
}

// wrap prints s word-wrapped so that prefix plus text fits the width,
// first before the first line and rest before the others. Widths are
// measured with the escapes taken out, so a bold word is as wide as a
// plain one. A word longer than the width goes on a line of its own.
func (r *renderer) wrap(s string, first, rest string) {
	for pi, para := range strings.Split(s, "\n") {
		prefix := rest
		if pi == 0 {
			prefix = first
		}
		avail := r.width - ansi.StringWidth(prefix)
		if avail < 10 {
			avail = 10
		}
		var line bytes.Buffer
		lineW := 0
		for _, word := range strings.Fields(para) {
			w := ansi.StringWidth(word)
			if lineW > 0 && lineW+1+w > avail {
				r.out.WriteString(prefix + line.String() + "\n")
				prefix = rest
				line.Reset()
				lineW = 0
			}
			if lineW > 0 {
				line.WriteByte(' ')
				lineW++
			}
			line.WriteString(word)
			lineW += w
		}
		r.out.WriteString(prefix + line.String() + "\n")
	}
}
