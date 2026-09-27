package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beatzball/swoop/internal/chat"
)

// askAPI streams an answer from an OpenAI-shaped API: chat completions
// for everyone, or OpenAI's Responses API when the provider wants their
// web search tool. Text goes to emit as it arrives; what the model is
// doing meanwhile (thinking, searching) goes to say.
//
// One small client in place of a library: the two request shapes are a
// few fields each, and the stream is lines of "data: {json}".
func askAPI(ctx context.Context, p provider, c *chat.Conversation, emit func(string), say func(string)) error {
	var body any
	path := "/chat/completions"
	if p.responses {
		path = "/responses"
		body = map[string]any{
			"model":  p.model,
			"input":  turns(c, "content"),
			"tools":  []map[string]string{{"type": "web_search"}},
			"stream": true,
		}
	} else {
		body = map[string]any{
			"model":    p.model,
			"messages": turns(c, "content"),
			"stream":   true,
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.url, "/")+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if p.key != "" {
		req.Header.Set("Authorization", "Bearer "+p.key)
	}
	client := &http.Client{Timeout: 0} // the context bounds it
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach %s: %w", p.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s answered %s: %s", p.url, resp.Status, apiMessage(msg))
	}
	return readStream(resp.Body, p.responses, emit, say)
}

// turns is the conversation as the API wants it: the turns with text,
// each a role and its content. The pending answer, empty, is left out.
func turns(c *chat.Conversation, field string) []map[string]string {
	var out []map[string]string
	for _, t := range c.Turns {
		if strings.TrimSpace(t.Text) == "" {
			continue
		}
		out = append(out, map[string]string{"role": t.Role, field: t.Text})
	}
	return out
}

// readStream reads "data: {json}" lines until [DONE] or the end. The two
// APIs put the text in different places; everything else is status.
func readStream(r io.Reader, responses bool, emit func(string), say func(string)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	thinking := false
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		if ev.Error != nil && ev.Error.Message != "" {
			return errors.New(ev.Error.Message)
		}
		if responses {
			switch {
			case ev.Type == "response.output_text.delta":
				emit(ev.Delta)
			case strings.HasPrefix(ev.Type, "response.web_search_call"):
				say("searching the web")
			case ev.Type == "response.failed":
				msg := "the model failed"
				if ev.Response != nil && ev.Response.Error != nil {
					msg = ev.Response.Error.Message
				}
				return errors.New(msg)
			}
			continue
		}
		for _, ch := range ev.Choices {
			if ch.Delta.Content != "" {
				emit(ch.Delta.Content)
			} else if (ch.Delta.Reasoning != "" || ch.Delta.ReasoningContent != "") && !thinking {
				thinking = true
				say("thinking")
			}
		}
	}
	return sc.Err()
}

// event is the union of what the two streams send, the fields swoop reads.
type event struct {
	// chat completions
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
	} `json:"choices"`
	// responses
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Response *struct {
		Error *apiError `json:"error"`
	} `json:"response"`
	// either
	Error *apiError `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
}

// apiMessage pulls the message out of an error body, or returns the body.
func apiMessage(body []byte) string {
	var e struct {
		Error *apiError `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != nil && e.Error.Message != "" {
		return e.Error.Message
	}
	s := strings.TrimSpace(string(body))
	if s == "" {
		return "no reason given"
	}
	return s
}

// listModels asks an OpenAI-shaped server what it has, for the Settings
// pane. A server that does not answer within a second has no models to
// offer.
func listModels(base, key string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/models", nil)
	if err != nil {
		return nil
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return nil
	}
	var ids []string
	for _, m := range out.Data {
		if strings.Contains(m.ID, "embed") {
			continue
		}
		ids = append(ids, m.ID)
	}
	return ids
}
