package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
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
	c.Worker = os.Getpid()
	_ = store.Save(c)
	p, err := resolve()
	if err != nil {
		return finish(store, c, fzf, "", "", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// What the transports write into, and the clock reads from.
	k := &sink{}
	if note := webNote(p); note != "" {
		// Said before the model says anything, and kept until it does.
		k.say(note)
	}
	done := make(chan struct{})
	go clock(store, c, fzf, k, done)

	var runErr error
	if p.api {
		runErr = askAPI(ctx, p, c, k.emit, k.say)
	} else {
		runErr = askCommand(ctx, p.line, c.Prompt(), k.emit, k.say)
	}
	close(done)
	answer, status := k.snapshot()
	if ctx.Err() == context.DeadlineExceeded {
		runErr = fmt.Errorf("no answer after %s", timeout)
	}
	text, failure := outcome(answer, status, runErr)
	if failure != nil && p.name != "" {
		failure = fmt.Errorf("%s: %w", p.name, failure)
	}
	return finish(store, c, fzf, text, "", failure)
}

// sink is where an answer gathers: the text so far and the last few
// status lines, under one lock, with a flag for the clock to know there
// is something new to draw.
type sink struct {
	mu     sync.Mutex
	text   strings.Builder
	status []string
	dirty  bool
}

func (k *sink) emit(s string) {
	k.mu.Lock()
	k.text.WriteString(s)
	k.dirty = true
	k.mu.Unlock()
}

// say records a status line: a tool being used, a warning, the reason
// something failed. Escape codes go, since ollama draws a spinner there,
// and an empty line is not a status.
func (k *sink) say(line string) {
	line = strings.TrimSpace(stripANSI(line))
	if line == "" {
		return
	}
	k.mu.Lock()
	k.status = append(k.status, line)
	if len(k.status) > 5 {
		k.status = k.status[1:]
	}
	k.dirty = true
	k.mu.Unlock()
}

func (k *sink) snapshot() (answer, status string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return stripANSI(k.text.String()), strings.Join(k.status, "\n")
}

// clock is the redraw rhythm: while the answer is still empty, a tick
// every 300 ms advances the dots; while text is arriving, a redraw at
// most every 100 ms, so a model that streams a token at a time does not
// run the preview command a hundred times a second.
func clock(store chat.Store, c *chat.Conversation, fzf redrawer, k *sink, done <-chan struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	n := 0
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			n++
			k.mu.Lock()
			waiting := k.text.Len() == 0
			redraw := k.dirty
			k.dirty = false
			if waiting && n%3 == 0 {
				c.Tick++
				redraw = true
			}
			if redraw {
				c.Turns[len(c.Turns)-1].Text = stripANSI(k.text.String())
				if len(k.status) > 0 {
					c.Status = k.status[len(k.status)-1]
				}
				_ = store.Save(c)
			}
			k.mu.Unlock()
			if redraw {
				fzf.post("refresh-preview")
			}
		}
	}
}

// askCommand runs a shell line with the prompt on its stdin: its stdout
// is the answer as it comes, its stderr lines are status.
func askCommand(ctx context.Context, line, prompt string, emit func(string), say func(string)) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", line)
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var stderrDone sync.WaitGroup
	stderrDone.Add(1)
	go func() {
		defer stderrDone.Done()
		sc := bufio.NewScanner(errPipe)
		sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
		for sc.Scan() {
			say(sc.Text())
		}
	}()
	buf := make([]byte, 4096)
	for {
		n, err := out.Read(buf)
		if n > 0 {
			emit(string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
	stderrDone.Wait()
	return cmd.Wait()
}

// ansi matches the escape sequences a command may write when it thinks
// it has a terminal: CSI sequences (cursor moves, erase, modes), OSC
// strings, and the two-byte kind. What is left is the text.
var ansi = regexp.MustCompile("\x1b\\[[0-9;?<=>]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(\x07|\x1b\\\\)|\x1b[@-Z\\\\^_]")

// citation matches the marks codex leaves in an answer for its own
// renderer, private-use characters around "cite…": not text.
var citation = regexp.MustCompile(`\x{e200}[^\x{e201}]*\x{e201}`)

// stripANSI takes the escape sequences out of s, the spinner glyphs
// ollama leaves behind with them, and the private-use marks codex
// writes for citations.
func stripANSI(s string) string {
	s = ansi.ReplaceAllString(s, "")
	s = citation.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r >= 0x2800 && r <= 0x28FF { // braille, the spinner's frames
			return -1
		}
		if r >= 0xE000 && r <= 0xF8FF { // private use: no model means these
			return -1
		}
		return r
	}, s)
}

// timeout is how long one answer may take. Long enough for a model that
// reads a few pages; short enough that a hung command is not "thinking"
// for the rest of the evening.
const timeout = 5 * time.Minute

// outcome decides what an ended command means. Text on stdout is the
// answer, and the exit status does not change that: a model that answered
// and then failed to exit cleanly still answered. No text is a failure
// whatever the exit status, because a pipeline's status is its last
// command's, and `claude | jq` exits 0 when claude fails; the reason is
// the last line of stderr, or the fact that nothing came.
func outcome(answer, stderr string, waitErr error) (text string, failure error) {
	if strings.TrimSpace(answer) != "" {
		return answer, nil
	}
	if msg := lastLine(stderr); msg != "" {
		return "", fmt.Errorf("%s", msg)
	}
	if waitErr != nil {
		return "", waitErr
	}
	return "", fmt.Errorf("the command printed nothing")
}

// finish writes the answer, or the reason there is none, and tells fzf to
// draw it and to list the conversations again.
func finish(store chat.Store, c *chat.Conversation, fzf redrawer, answer, _ string, err error) error {
	c.Turns[len(c.Turns)-1].Text = strings.TrimSpace(answer)
	c.Pending = false
	c.Status = ""
	c.Worker = 0
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
