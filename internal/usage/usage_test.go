package usage

import (
	"strings"
	"testing"
	"time"

	"github.com/beatzball/swoop/internal/protocol"
)

func TestRecordLoadStatsRecent(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, id := range []string{"/Applications/Safari.app", "ext/system/lock", "/Applications/Safari.app"} {
		if err := Record(id, "app", "Safari"); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := Load()
	if err != nil || len(entries) != 3 {
		t.Fatalf("load: %d %v", len(entries), err)
	}
	stats := Stats(entries)
	if len(stats) != 2 || stats[0].ID != "/Applications/Safari.app" || stats[0].Count != 2 || stats[1].Count != 1 {
		t.Fatalf("stats: %+v", stats)
	}
	if got := Recent(entries, 5); len(got) != 2 || got[0] != "/Applications/Safari.app" || got[1] != "ext/system/lock" {
		t.Fatalf("recent, newest first, each once: %v", got)
	}
	if err := Clear(); err != nil {
		t.Fatal(err)
	}
	if entries, _ := Load(); len(entries) != 0 {
		t.Fatalf("after clear: %d", len(entries))
	}
	if err := Clear(); err != nil {
		t.Fatalf("clearing twice is fine: %v", err)
	}
}

func TestFrontPutsRecentFirstOnce(t *testing.T) {
	items := []protocol.Item{
		{ID: "a", Title: "Activity Monitor", Subtitle: ""},
		{ID: "b", Title: "Books"},
		{ID: "c", Title: "Chess"},
		{ID: "d", Title: "Dictionary", Subtitle: "look up"},
	}
	out := Front(items, []string{"d", "zzz", "b"}, 5)
	ids := make([]string, len(out))
	for i, it := range out {
		ids[i] = it.ID
	}
	if strings.Join(ids, "") != "dbac" {
		t.Fatalf("order: %v", ids)
	}
	if !strings.Contains(out[0].Subtitle, "look up") || !strings.Contains(out[0].Subtitle, Mark) {
		t.Fatalf("a recent row keeps its subtitle and gains the mark: %q", out[0].Subtitle)
	}
	if !strings.Contains(out[1].Subtitle, Mark) {
		t.Fatalf("an empty subtitle becomes the mark: %q", out[1].Subtitle)
	}
	if out := Front(items, []string{"d", "b", "c"}, 2); len(out) != 4 || out[0].ID != "d" || out[1].ID != "b" || out[2].ID != "a" {
		t.Fatalf("n caps the group: %+v", out)
	}
	if out := Front(items, nil, 5); out[0].ID != "a" {
		t.Fatal("no log, no change")
	}
}

func TestStatsOrderAndTimes(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{Time: now.Add(-3 * time.Hour), ID: "x", Title: "X"},
		{Time: now.Add(-2 * time.Hour), ID: "y", Title: "Y"},
		{Time: now.Add(-1 * time.Hour), ID: "y", Title: "Y"},
	}
	s := Stats(entries)
	if s[0].ID != "y" || s[0].Count != 2 || !s[0].First.Equal(now.Add(-2*time.Hour)) || !s[0].Last.Equal(now.Add(-1*time.Hour)) {
		t.Fatalf("%+v", s[0])
	}
}
