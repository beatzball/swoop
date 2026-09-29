package models

import (
	"reflect"
	"testing"
)

func TestParseOllamaList(t *testing.T) {
	out := "NAME                    ID              SIZE      MODIFIED\n" +
		"qwen3:8b                500a1f067a9f    5.2 GB    2 hours ago\n" +
		"nomic-embed-text:latest 0a109f422b47    274 MB    3 days ago\n" +
		"llama3.2:latest         a80c4f17acd5    2.0 GB    2 weeks ago\n"
	got := parseOllamaList(out)
	want := []string{"qwen3:8b", "llama3.2:latest"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q: in order, embeddings left out", got, want)
	}
	if got := parseOllamaList("NAME ID SIZE MODIFIED\n"); len(got) != 0 {
		t.Fatalf("no models: %q", got)
	}
}
