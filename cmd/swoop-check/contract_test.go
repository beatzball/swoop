package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// The document and its copy for the site, from this package's folder.
const (
	document = "../../CONTRACT.md"
	sitePage = "../../site/content/docs/contract.md"
)

// update rewrites the site's page from the document:
//
//	go test ./cmd/swoop-check -run TestSitePageIsTheDocument -update
var update = flag.Bool("update", false, "rewrite the site's contract page from CONTRACT.md")

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The version this program checks against is the one the document says
// at its top. A new version changes both.
func TestVersionIsTheDocuments(t *testing.T) {
	m := regexp.MustCompile(`(?m)^\*\*contract ([0-9.]+)\*\*`).FindStringSubmatch(read(t, document))
	if m == nil {
		t.Fatal("CONTRACT.md has no **contract N** line at its top")
	}
	if m[1] != contract {
		t.Errorf("CONTRACT.md is contract %s, swoop-check checks contract %s", m[1], contract)
	}
}

// Every rule a finding can carry is in the document's table, with what
// it asks, and the table names no rule the program has not got.
func TestRulesAreTheDocuments(t *testing.T) {
	_, section, found := strings.Cut(read(t, document), "\n## swoop-check\n")
	if !found {
		t.Fatal("CONTRACT.md has no swoop-check section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	var listed []string
	for _, m := range regexp.MustCompile("(?m)^\\| `([a-z-]+)` \\|").FindAllStringSubmatch(section, -1) {
		listed = append(listed, m[1])
	}
	if !slices.Equal(listed, rules) {
		t.Errorf("CONTRACT.md lists the rules\n%v\nswoop-check has\n%v", listed, rules)
	}
}

// example is the first shell program in the document: the extension in
// ten lines.
func example(t *testing.T) string {
	t.Helper()
	_, rest, found := strings.Cut(read(t, document), "```sh\n#!/bin/sh\n")
	if !found {
		t.Fatal("CONTRACT.md has no shell program in a sh block")
	}
	body, _, found := strings.Cut(rest, "```")
	if !found {
		t.Fatal("the example's block does not end")
	}
	return "#!/bin/sh\n" + body
}

// The document says a reader with it alone can write a working
// extension, and gives one in ten lines. It is ten lines, it meets the
// contract with every verb tried, and it does what the text beside it
// says.
func TestExampleInTheDocumentRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts do not run as programs on Windows")
	}
	text := example(t)
	if n := strings.Count(text, "\n"); n != 10 {
		t.Errorf("the example is %d lines; the document calls it ten", n)
	}
	dir := filepath.Join(t.TempDir(), "hello")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hello"), []byte(text), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, found, untried := check(dir, patient, true); len(found) != 0 || len(untried) != 0 {
		t.Fatalf("the example does not meet the contract: %v, not tried %v", found, untried)
	}
	c := &checker{limit: patient, env: os.Environ()}
	var err error
	if c.e, err = find(dir); err != nil {
		t.Fatal(err)
	}
	if a := c.ask(patient, "run", "fr"); a.err != nil || string(a.out) != "greeting: fr\n" {
		t.Errorf("hello run fr printed %q, %v; the document says greeting: fr", a.out, a.err)
	}
}

// A link inside the document goes to a heading that is there, by the
// name GitHub and the site both give it.
func TestLinksInsideTheDocument(t *testing.T) {
	text := read(t, document)
	slugs := map[string]bool{}
	strip := regexp.MustCompile(`[^\w\s-]`)
	for _, m := range regexp.MustCompile(`(?m)^#{2,4} (.+)$`).FindAllStringSubmatch(text, -1) {
		slug := strings.Join(strings.Fields(strip.ReplaceAllString(strings.ToLower(m[1]), "")), "-")
		if slugs[slug] {
			t.Errorf("two headings are both #%s; a link to it would be to the first only", slug)
		}
		slugs[slug] = true
	}
	for _, m := range regexp.MustCompile(`\]\(#([^)]+)\)`).FindAllStringSubmatch(text, -1) {
		if !slugs[m[1]] {
			t.Errorf("a link goes to #%s, and no heading has that name", m[1])
		}
	}
}

// The site's page is the document: the same text under the site's own
// front matter, with the document's title line left to the front matter.
// The site is built with site/ alone, so the page is a file there, and
// this test is what keeps it the same.
func TestSitePageIsTheDocument(t *testing.T) {
	title, body, found := strings.Cut(read(t, document), "\n\n")
	if !found || !strings.HasPrefix(title, "# ") {
		t.Fatal("CONTRACT.md does not start with a title line and a blank line")
	}
	page := read(t, sitePage)
	parts := strings.SplitN(page, "---\n", 3)
	if len(parts) != 3 || parts[0] != "" {
		t.Fatal("the site's page does not start with front matter between --- lines")
	}
	want := "\n" + body
	if parts[2] == want {
		return
	}
	if !*update {
		t.Fatalf("%s is not CONTRACT.md's text. Change CONTRACT.md, then run\n  go test ./cmd/swoop-check -run TestSitePageIsTheDocument -update", sitePage)
	}
	if err := os.WriteFile(sitePage, []byte("---\n"+parts[1]+"---\n"+want), 0o644); err != nil {
		t.Fatal(err)
	}
}
