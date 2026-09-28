package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSkipsCommentsAndBadLines(t *testing.T) {
	links := parse([]byte("# header\n\nGoogle\thttps://g/?q={argument}\nbad line\n Docs \t /tmp/Docs \tFinder\n"))
	if len(links) != 2 {
		t.Fatalf("want 2 links, got %d: %+v", len(links), links)
	}
	if links[0].Name != "Google" || !links[0].HasArgument() {
		t.Errorf("first link wrong: %+v", links[0])
	}
	if links[1].Name != "Docs" || links[1].Link != "/tmp/Docs" || links[1].App != "Finder" {
		t.Errorf("fields not trimmed: %+v", links[1])
	}
}

func TestFillEncodesWebOnly(t *testing.T) {
	web := Link{Link: "https://x.test/?q={argument}"}
	if got := web.Fill("hello world & co"); got != "https://x.test/?q=hello+world+%26+co" {
		t.Errorf("web fill: %q", got)
	}
	opts := Link{Link: `https://x.test/?q={argument name="term"}`}
	if got := opts.Fill("a"); got != "https://x.test/?q=a" {
		t.Errorf("argument with options: %q", got)
	}
	file := Link{Link: "/notes/{argument}.md"}
	if got := file.Fill("to do"); got != "/notes/to do.md" {
		t.Errorf("file fill: %q", got)
	}
	plain := Link{Link: "https://x.test/"}
	if plain.HasArgument() || plain.Fill("a") != "https://x.test/" {
		t.Errorf("a link without {argument} must not change")
	}
}

func TestWhereIsHostForWeb(t *testing.T) {
	if got := (Link{Link: "https://en.wikipedia.org/w/index.php?search={argument}"}).Where(); got != "en.wikipedia.org" {
		t.Errorf("host: %q", got)
	}
	if got := (Link{Link: "obsidian://open?vault=me"}).Where(); got != "obsidian://open?vault=me" {
		t.Errorf("deeplink: %q", got)
	}
}

func TestDefaultsParse(t *testing.T) {
	links := parse([]byte(defaults))
	if len(links) != 5 {
		t.Fatalf("want 5 defaults, got %d", len(links))
	}
	for _, l := range links {
		if !l.HasArgument() {
			t.Errorf("%s ships without {argument}", l.Name)
		}
	}
}

// add on a fresh config starts the file from the defaults and keeps the
// new link; a second add of the same name replaces it; remove takes it
// out and keeps the rest.
func TestAddRemoveRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := add(Link{Name: "Docs", Link: "https://docs.test/"}); err != nil {
		t.Fatal(err)
	}
	links, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 6 || links[5].Name != "Docs" {
		t.Fatalf("after add: %+v", links)
	}
	if err := add(Link{Name: "Docs", Link: "https://docs.test/v2", App: "Safari"}); err != nil {
		t.Fatal(err)
	}
	links, _ = load()
	if len(links) != 6 || links[5].Link != "https://docs.test/v2" || links[5].App != "Safari" {
		t.Fatalf("after replace: %+v", links)
	}
	if err := remove("Google"); err != nil {
		t.Fatal(err)
	}
	links, _ = load()
	if len(links) != 5 || links[0].Name != "DuckDuckGo" {
		t.Fatalf("after remove: %+v", links)
	}
	data, _ := os.ReadFile(path())
	if string(data[:1]) != "#" {
		t.Errorf("the file lost its header: %q", data)
	}
}

func TestImportJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	file := filepath.Join(t.TempDir(), "export.json")
	body := `[{"name":"Maps","link":"https://maps.test/?q={argument}","iconName":"map","openWith":"/Applications/Safari.app"},{"name":"","link":"x"}]`
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := importJSON(file); err != nil {
		t.Fatal(err)
	}
	links, _ := load()
	l, ok := find(links, "Maps")
	if !ok || l.App != "/Applications/Safari.app" || !l.HasArgument() {
		t.Fatalf("import: %+v", links)
	}
	if err := os.WriteFile(file, []byte(`{"not":"an array"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := importJSON(file); err == nil {
		t.Error("an object, not an array, should be an error")
	}
}
