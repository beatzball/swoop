package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beatzball/swoop/internal/chat"
	"github.com/beatzball/swoop/internal/settings"
)

func TestReadStreamChatCompletions(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"choices":[{"delta":{"role":"assistant","reasoning_content":"hmm"}}]}`,
		`data: {"choices":[{"delta":{"reasoning_content":"..."}}]}`,
		`data: {"choices":[{"delta":{"content":"Blue"}}]}`,
		``,
		`data: {"choices":[{"delta":{"content":" sky."}}]}`,
		`data: [DONE]`,
	}, "\n")
	var text, status []string
	err := readStream(strings.NewReader(stream), false, func(s string) { text = append(text, s) }, func(s string) { status = append(status, s) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(text, "") != "Blue sky." {
		t.Fatalf("text: %q", text)
	}
	if len(status) != 1 || status[0] != "thinking" {
		t.Fatalf("reasoning is one status line, not text: %q", status)
	}
}

func TestReadStreamResponses(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"type":"response.created"}`,
		`data: {"type":"response.web_search_call.in_progress"}`,
		`data: {"type":"response.output_text.delta","delta":"It is "}`,
		`data: {"type":"response.output_text.delta","delta":"96°F."}`,
		`data: {"type":"response.completed"}`,
	}, "\n")
	var text, status []string
	if err := readStream(strings.NewReader(stream), true, func(s string) { text = append(text, s) }, func(s string) { status = append(status, s) }); err != nil {
		t.Fatal(err)
	}
	if strings.Join(text, "") != "It is 96°F." || len(status) != 1 || status[0] != "searching the web" {
		t.Fatalf("text %q status %q", text, status)
	}
	failed := `data: {"type":"response.failed","response":{"error":{"message":"quota"}}}` + "\n"
	if err := readStream(strings.NewReader(failed), true, func(string) {}, func(string) {}); err == nil || err.Error() != "quota" {
		t.Fatalf("a failed response is an error with its message: %v", err)
	}
}

func TestAskAPIAgainstAFakeServer(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b := make([]byte, 4096)
		n, _ := r.Body.Read(b)
		gotBody = string(b[:n])
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n"))
	}))
	defer srv.Close()
	c := &chat.Conversation{}
	c.Ask("hello")
	var text string
	p := provider{api: true, url: srv.URL + "/v1", model: "m", key: "k"}
	if err := askAPI(context.Background(), p, c, func(s string) { text += s }, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if text != "hi" || gotAuth != "Bearer k" || gotPath != "/v1/chat/completions" {
		t.Fatalf("text %q auth %q path %q", text, gotAuth, gotPath)
	}
	if !strings.Contains(gotBody, `"model":"m"`) || !strings.Contains(gotBody, `"content":"hello"`) || !strings.Contains(gotBody, `"stream":true`) {
		t.Fatalf("body: %s", gotBody)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer bad.Close()
	p.url = bad.URL + "/v1"
	err := askAPI(context.Background(), p, c, func(string) {}, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("a refused request says why: %v", err)
	}
}

func TestResolvePresets(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	set := func(k, v string) {
		t.Helper()
		if err := settings.Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	set("ai", "lmstudio:gemma")
	p, err := resolve()
	if err != nil || !p.api || p.url != lmStudioURL || p.model != "gemma" || p.canSearch {
		t.Fatalf("lmstudio: %+v %v", p, err)
	}
	set("ai", "openai:gpt-5")
	if _, err := resolve(); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("openai without a key: %v", err)
	}
	set("ai_key", "sk-test")
	set("web", "on")
	p, err = resolve()
	if err != nil || !p.api || !p.responses || !p.canSearch || p.url != openAIURL {
		t.Fatalf("openai with web on uses the Responses API: %+v %v", p, err)
	}
	set("ai_url", "https://openrouter.ai/api/v1")
	p, _ = resolve()
	if p.responses || p.canSearch || webNote(p) == "" {
		t.Fatalf("another host: chat completions, and the note that it cannot search: %+v", p)
	}
	set("ai", "some command of mine")
	p, _ = resolve()
	if p.api || p.line != "some command of mine" {
		t.Fatalf("a line of your own: %+v", p)
	}
	if onPath("ollama") {
		set("ai", "ollama:x")
		p, _ = resolve()
		if p.line == "" || !strings.Contains(p.line, "--think=false") || webNote(p) == "" {
			t.Fatalf("ollama: %+v", p)
		}
	}
}
