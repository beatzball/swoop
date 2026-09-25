package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/beatzball/swoop/internal/chat"
)

// work is the detached worker for one conversation: run the model with
// the conversation on its stdin, and put its stdout into the file as it
// comes, telling fzf to redraw the preview along the way.
//
// Three rhythms: while the answer is still empty, a tick every 300 ms
// advances the dots; while text is arriving, a redraw at most every 100
// ms, so a model that streams a token at a time does not run the preview
// command a hundred times a second; and one reload of the rows at the
// end, so the subtitle stops saying "thinking".
func work(store chat.Store, id string) error {
	c, err := store.Load(id)
	if err != nil {
		return err
	}
	if !c.Pending {
		return nil
	}
	fzf := newRedrawer()
	line, err := commandLine()
	if err != nil {
		return finish(store, c, fzf, "", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", line)
	cmd.Stdin = strings.NewReader(c.Prompt())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return finish(store, c, fzf, "", err)
	}
	if err := cmd.Start(); err != nil {
		return finish(store, c, fzf, "", err)
	}

	var mu sync.Mutex
	var text strings.Builder
	dirty := false
	done := make(chan struct{})

	// The clock: dots while nothing has arrived, redraws while it is.
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		n := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				n++
				mu.Lock()
				waiting := text.Len() == 0
				redraw := dirty
				dirty = false
				if waiting && n%3 == 0 {
					c.Tick++
					redraw = true
				}
				if redraw {
					c.Turns[len(c.Turns)-1].Text = text.String()
					_ = store.Save(c)
				}
				mu.Unlock()
				if redraw {
					fzf.post("refresh-preview")
				}
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := out.Read(buf)
		if n > 0 {
			mu.Lock()
			text.Write(buf[:n])
			dirty = true
			mu.Unlock()
		}
		if err != nil {
			if err != io.EOF {
				cancel()
			}
			break
		}
	}
	waitErr := cmd.Wait()
	close(done)
	mu.Lock()
	answer := text.String()
	mu.Unlock()
	if waitErr != nil && strings.TrimSpace(answer) == "" {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = waitErr.Error()
		}
		return finish(store, c, fzf, "", fmt.Errorf("%s", lastLine(msg)))
	}
	return finish(store, c, fzf, answer, nil)
}

// finish writes the answer, or the reason there is none, and tells fzf to
// draw it and to list the conversations again.
func finish(store chat.Store, c *chat.Conversation, fzf redrawer, answer string, err error) error {
	c.Turns[len(c.Turns)-1].Text = strings.TrimSpace(answer)
	c.Pending = false
	if err != nil {
		c.Error = "The AI command failed: " + err.Error()
	}
	saveErr := store.Save(c)
	fzf.post("refresh-preview+reload(swoop-nav rows)")
	return saveErr
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

// redrawer posts actions to the fzf that opened the pane, through the
// socket or port fzf exported to its children. With neither there is no
// fzf to tell, and posts are dropped: the file still fills, and the next
// cursor move shows it.
type redrawer struct {
	sock, port, key string
	http            *http.Client
}

func newRedrawer() redrawer {
	f := redrawer{sock: os.Getenv("FZF_SOCK"), port: os.Getenv("FZF_PORT"), key: os.Getenv("FZF_API_KEY")}
	transport := &http.Transport{}
	if f.sock != "" {
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", f.sock)
		}
	}
	f.http = &http.Client{Transport: transport, Timeout: 2 * time.Second}
	return f
}

func (f redrawer) post(action string) {
	if f.sock == "" && f.port == "" {
		return
	}
	host := "127.0.0.1:" + f.port
	if f.sock != "" {
		host = "fzf"
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+host+"/", strings.NewReader(action))
	if err != nil {
		return
	}
	if f.key != "" {
		req.Header.Set("x-api-key", f.key)
	}
	resp, err := f.http.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
