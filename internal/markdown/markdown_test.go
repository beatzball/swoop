package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// plain is the rendering with the escapes taken out: what the eye lines
// up, without the colours.
func plain(md string, width int) string {
	return ansi.Strip(Render(md, width))
}

func TestHeadingsAndEmphasis(t *testing.T) {
	out := Render("# Title\n\nSome **bold** and *italic* and `code`.\n", 60)
	for _, want := range []string{bold + underline + "Title", bold + "bold" + unbold, italic + "italic" + unitalic, codeFg + "code" + reset} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	if p := plain("# Title\n\ntext\n", 60); p != "Title\n\ntext\n" {
		t.Errorf("blocks are separated by one blank line: %q", p)
	}
}

func TestParagraphsWrapToTheWidth(t *testing.T) {
	md := "one two three four five six seven eight nine ten eleven twelve"
	out := plain(md, 24)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if ansi.StringWidth(line) > 24 {
			t.Errorf("line wider than 24: %q", line)
		}
	}
	if strings.Count(out, "\n") < 3 {
		t.Errorf("should wrap into several lines: %q", out)
	}
	if plain("**bold** word", 60) != "bold word\n" {
		t.Errorf("styles do not count toward width or add spaces: %q", plain("**bold** word", 60))
	}
}

func TestLists(t *testing.T) {
	out := plain("- apple\n- banana\n  - nested\n\n1. first\n2. second\n", 60)
	want := "• apple\n• banana\n  • nested\n\n1. first\n2. second\n"
	if out != want {
		t.Errorf("got\n%q\nwant\n%q", out, want)
	}
	// A wrapped item keeps its text under the text, not under the bullet.
	out = plain("- aaa bbb ccc ddd eee fff ggg hhh", 20)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "• aaa") || !strings.HasPrefix(lines[1], "  ") {
		t.Errorf("wrapped item indent: %q", out)
	}
}

func TestQuoteCodeRuleAndLink(t *testing.T) {
	if p := plain("> quoted words\n", 60); p != "│ quoted words\n" {
		t.Errorf("quote: %q", p)
	}
	if p := plain("```go\nfmt.Println(1)\n```\n", 60); p != "  fmt.Println(1)\n" {
		t.Errorf("code block indented, not highlighted: %q", p)
	}
	if p := plain("a\n\n---\n\nb\n", 30); !strings.Contains(p, strings.Repeat("─", 30)) {
		t.Errorf("rule spans the width: %q", p)
	}
	if p := plain("see [the docs](https://x.test/d) now", 60); p != "see the docs (https://x.test/d) now\n" {
		t.Errorf("link is its text with the address after: %q", p)
	}
	if p := plain("<https://x.test>", 60); p != "https://x.test\n" {
		t.Errorf("autolink: %q", p)
	}
}

func TestTable(t *testing.T) {
	md := "| when | temp |\n|---|---|\n| now | 96°F |\n| later | 80°F |\n"
	out := plain(md, 60)
	want := "when  │ temp\n──────┼─────\nnow   │ 96°F\nlater │ 80°F\n"
	if out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
	if !strings.Contains(Render(md, 60), bold+"when") {
		t.Error("the header row is bold")
	}
}

func TestNarrowWidthIsClamped(t *testing.T) {
	out := plain("a b c d e f g h i j k l m n o p", 5)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if ansi.StringWidth(line) > 20 {
			t.Errorf("line wider than the clamp: %q", line)
		}
	}
}
