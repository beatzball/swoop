//go:build ignore

// gen writes table.tsv, the emoji and symbols swoop-emoji searches, from
// Unicode's and CLDR's own files. It is run by hand when a new Unicode
// version is out, not at build time, so building the kit needs no network:
//
//	curl -LO https://unicode.org/Public/emoji/latest/emoji-test.txt
//	curl -LO https://unicode.org/Public/UCD/latest/ucd/UnicodeData.txt
//	curl -L -o en.xml https://raw.githubusercontent.com/unicode-org/cldr/main/common/annotations/en.xml
//	curl -L -o en-derived.xml https://raw.githubusercontent.com/unicode-org/cldr/main/common/annotationsDerived/en.xml
//	go run gen.go emoji-test.txt UnicodeData.txt en.xml en-derived.xml > table.tsv
//
// One line per character: the character, its name, its group, its tone
// template, and its keywords joined by "|". The tone template is the
// character with ~ where a skin-tone modifier goes, empty when it takes
// none; see tone in table.go.
package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type row struct {
	char, name, group, tone string
	keywords                []string
}

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: go run gen.go emoji-test.txt UnicodeData.txt en.xml en-derived.xml")
		os.Exit(2)
	}
	names, keywords := annotations(os.Args[3])
	dn, dk := annotations(os.Args[4])
	for k, v := range dn {
		if _, ok := names[k]; !ok {
			names[k] = v
		}
	}
	for k, v := range dk {
		if _, ok := keywords[k]; !ok {
			keywords[k] = v
		}
	}
	rows, seen := emoji(os.Args[1], keywords)
	rows = append(rows, symbols(os.Args[2], names, keywords, seen)...)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.char, r.name, r.group, r.tone, strings.Join(r.keywords, "|"))
	}
}

// strip drops the emoji presentation selector, which CLDR leaves out of
// its keys and emoji-test.txt keeps in the fully-qualified form.
func strip(s string) string { return strings.ReplaceAll(s, "️", "") }

// annotations reads a CLDR annotations file: the spoken name (type="tts")
// and the keywords of each character, keyed without FE0F.
func annotations(file string) (map[string]string, map[string][]string) {
	f, err := os.Open(file)
	check(err)
	defer f.Close()
	var doc struct {
		Items []struct {
			CP   string `xml:"cp,attr"`
			Type string `xml:"type,attr"`
			Text string `xml:",chardata"`
		} `xml:"annotations>annotation"`
	}
	check(xml.NewDecoder(f).Decode(&doc))
	names := map[string]string{}
	keywords := map[string][]string{}
	for _, a := range doc.Items {
		key := strip(a.CP)
		if a.Type == "tts" {
			names[key] = strings.TrimSpace(a.Text)
			continue
		}
		for _, k := range strings.Split(a.Text, "|") {
			if k = strings.ToLower(strings.TrimSpace(k)); k != "" {
				keywords[key] = append(keywords[key], k)
			}
		}
	}
	return names, keywords
}

var testLine = regexp.MustCompile(`^([0-9A-F ]+?)\s*; fully-qualified\s*# \S+ E[0-9.]+ (.+)$`)

var tones = []string{"light skin tone", "medium-light skin tone", "medium skin tone", "medium-dark skin tone", "dark skin tone"}

// emoji reads every fully-qualified emoji in emoji-test.txt, in its order,
// which is the order people know from a keyboard's picker. A variant with
// one skin tone is not a row of its own: it becomes the base's tone
// template. Variants with two tones (a couple, each with their own) are
// left out; the template gives both people the same tone.
func emoji(file string, keywords map[string][]string) ([]row, map[string]bool) {
	f, err := os.Open(file)
	check(err)
	defer f.Close()
	var rows []row
	index := map[string]int{} // name to row
	seen := map[string]bool{}
	variants := map[string][]string{} // base name to its one-tone forms, light to dark
	group := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if g, ok := strings.CutPrefix(line, "# group: "); ok {
			group = g
			continue
		}
		m := testLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		char := decode(m[1])
		seen[strip(char)] = true
		name := m[2]
		if base, tone, ok := strings.Cut(name, ": "); ok && isTone(tone) {
			variants[base] = append(variants[base], char)
			if tone == tones[0] {
				if i, found := index[base]; found {
					rows[i].tone = strings.ReplaceAll(char, "\U0001F3FB", "~")
				}
			}
			continue
		}
		if strings.Contains(name, "skin tone") {
			continue
		}
		index[name] = len(rows)
		rows = append(rows, row{char: char, name: name, group: group, keywords: keywords[strip(char)]})
	}
	check(sc.Err())
	// The template must give back exactly the five forms Unicode lists;
	// one that does not is a change in the file this code has not seen.
	for _, r := range rows {
		if r.tone == "" {
			continue
		}
		for k, want := range variants[r.name] {
			if got := strings.ReplaceAll(r.tone, "~", string(rune(0x1F3FB+k))); got != want {
				check(fmt.Errorf("%s: tone %d gives %q, Unicode lists %q", r.name, k+1, got, want))
			}
		}
	}
	return rows, seen
}

func isTone(s string) bool {
	for _, t := range tones {
		if s == t {
			return true
		}
	}
	return false
}

func decode(hex string) string {
	var b strings.Builder
	for _, h := range strings.Fields(hex) {
		n, err := strconv.ParseUint(h, 16, 32)
		check(err)
		b.WriteRune(rune(n))
	}
	return b.String()
}

// The symbols, by group: the ranges and characters people reach for when
// they need one they cannot type. Anything that is also an emoji is left
// to the emoji rows.
var symbolGroups = []struct {
	group string
	word  string // a keyword every symbol in the group gets
	runes []rune
}{
	{"Arrows", "arrow", span(0x2190, 0x21FF)},
	{"Math", "math", append([]rune("±×÷≈≠≤≥∞√∛∑∏∫∮∂∆∇∈∉∋∩∪⊂⊃⊆⊇∀∃∄∅¬∧∨⊕⊗⊙∝∠∡⊥∥∦≡≢≅∼≃≪≫∘∙⋅∴∵∶∷′″‴°‰‱ℕℤℚℝℂℍℙℵℏ⌈⌉⌊⌋⟨⟩⟦⟧∣∤∓⊢⊣⊤⊨"), span(0x2070, 0x2079)...)},
	{"Fractions and numbers", "number", []rune("¼½¾⅓⅔⅕⅖⅗⅘⅙⅚⅛⅜⅝⅞¹²³⁰⁴⁵⁶⁷⁸⁹₀₁₂₃₄₅₆₇₈₉ⅠⅡⅢⅣⅤⅥⅦⅧⅨⅩⅪⅫ①②③④⑤⑥⑦⑧⑨⑩")},
	{"Currency", "currency", append([]rune("$¢£¤¥ƒ"), span(0x20A0, 0x20C0)...)},
	{"Punctuation", "punctuation", []rune("–—‒―…•‣·‘’‚‛“”„‟‹›«»†‡§¶©®™℠‽⁂¡¿¦¨¯´¸ªº№℗℮⁓⁕※⁘⁙⁚⁛⁜⁝⁞")},
	{"Keyboard", "key", []rune("⌘⌥⇧⌃⎋⏎⌫⌦⇥⇤⌤␣⌧⌨⎆⎇⏏⇪⌽⎌")},
	{"Shapes", "shape", append(span(0x25A0, 0x25FF), []rune("★☆✓✔✗✘✦✧♠♡♢♣♤♥♦♧♩♪♫♬☐☑☒")...)},
	{"Box drawing", "box", span(0x2500, 0x257F)},
	{"Greek", "greek", append(span(0x0391, 0x03A9), span(0x03B1, 0x03C9)...)},
}

// extra are the words a keyboard symbol goes by, which neither Unicode's
// name ("place of interest sign") nor CLDR's says.
var extra = map[rune][]string{
	'⌘': {"command", "cmd"}, '⌥': {"option", "alt"}, '⇧': {"shift"}, '⌃': {"control", "ctrl"},
	'⎋': {"escape", "esc"}, '⏎': {"return", "enter"}, '⌫': {"delete", "backspace"},
	'⌦': {"forward delete"}, '⇥': {"tab"}, '⇤': {"backtab"}, '␣': {"space"},
	'⇪': {"caps lock"}, '⏏': {"eject"},
}

func span(from, to rune) []rune {
	var rs []rune
	for r := from; r <= to; r++ {
		rs = append(rs, r)
	}
	return rs
}

// symbols makes a row for each curated symbol that Unicode names and that
// is not an emoji. Its name is CLDR's spoken name when it has one ("euro"),
// Unicode's otherwise ("box drawings light horizontal"); Unicode's name
// is kept among the keywords either way, so "rightwards" finds →.
func symbols(file string, names map[string]string, keywords map[string][]string, seen map[string]bool) []row {
	f, err := os.Open(file)
	check(err)
	defer f.Close()
	ucd := map[rune]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Split(sc.Text(), ";")
		n, err := strconv.ParseUint(fields[0], 16, 32)
		if err != nil || strings.HasPrefix(fields[1], "<") {
			continue
		}
		ucd[rune(n)] = strings.ToLower(fields[1])
	}
	check(sc.Err())
	var rows []row
	done := map[rune]bool{}
	for _, g := range symbolGroups {
		for _, r := range g.runes {
			s := string(r)
			uname, ok := ucd[r]
			if !ok || seen[s] || done[r] {
				continue
			}
			done[r] = true
			name := uname
			if n, ok := names[s]; ok {
				name = n
			}
			kw := append(append([]string{g.word}, keywords[s]...), extra[r]...)
			if name != uname {
				kw = append(kw, uname)
			}
			rows = append(rows, row{char: s, name: name, group: g.group, keywords: dedupe(kw)})
		}
	}
	return rows
}

func dedupe(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}
