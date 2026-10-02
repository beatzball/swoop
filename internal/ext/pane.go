package ext

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/beatzball/swoop/internal/settings"
)

// KeyFile names the file beside an extension's executable that holds the
// keys it claims, one claim per line: the key, the id of the view it
// opens, and the view's title, the pane's prompt.
//
//	tab ask Ask AI
//
// A file, as the keyword is, so the launcher reads it without running
// anything. A blank line and a line that starts with # are skipped.
const KeyFile = "key"

// Claim is one line of a key file.
type Claim struct {
	Key   string // fzf's name for the key: "tab", "alt-,"
	View  string // the view's id, in the extension's own terms
	Title string // the pane's prompt; the extension's name without one
}

// Claims returns the extension's claims, in the file's order. A key an
// extension may not claim is still returned, so the caller can say so;
// ByKey and Keys leave it out.
func (e Extension) Claims() []Claim {
	data, err := os.ReadFile(filepath.Join(e.Dir, KeyFile))
	if err != nil {
		return nil
	}
	var claims []Claim
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		c := Claim{Key: fields[0], View: fields[1], Title: strings.Join(fields[2:], " ")}
		if c.Title == "" {
			c.Title = e.Name
		}
		claims = append(claims, c)
	}
	return claims
}

// ValidKey says whether an extension may claim key: tab, shift-tab, f1 to
// f12, and alt with one lower-case letter, digit, comma, full stop or
// slash. The rest belong to the launcher (Enter, Esc, the arrows, ctrl-k)
// or to editing the bar (the ctrl keys). The set is also what makes a
// key's name safe to put, as it is, in a binding and in a shell command.
func ValidKey(key string) bool {
	switch key {
	case "tab", "shift-tab":
		return true
	}
	if n, ok := strings.CutPrefix(key, "f"); ok {
		i, err := strconv.Atoi(n)
		return err == nil && i >= 1 && i <= 12 && n == strconv.Itoa(i)
	}
	c, ok := strings.CutPrefix(key, "alt-")
	if !ok || len(c) != 1 {
		return false
	}
	return c[0] >= 'a' && c[0] <= 'z' || c[0] >= '0' && c[0] <= '9' || strings.Contains(",./", c)
}

// ByKey returns the extension that has key, and its claim. When two
// claim the same key, the first by name wins, as with a keyword; exts is
// in that order from Discover.
func ByKey(exts []Extension, key string) (Extension, Claim, bool) {
	if !ValidKey(key) {
		return Extension{}, Claim{}, false
	}
	for _, e := range exts {
		for _, c := range e.Claims() {
			if c.Key == key {
				return e, c, true
			}
		}
	}
	return Extension{}, Claim{}, false
}

// Keys returns every key the extensions claim, each once.
func Keys(exts []Extension) []string {
	seen := map[string]bool{}
	var keys []string
	for _, e := range exts {
		for _, c := range e.Claims() {
			if ValidKey(c.Key) && !seen[c.Key] {
				seen[c.Key] = true
				keys = append(keys, c.Key)
			}
		}
	}
	return keys
}

// KeyReport says who has which key, one line per key, and what is wrong
// with the claims that lost: a key two extensions claim, a key nobody may
// claim. It is what `swoop status` prints, so a key that does not do what
// its extension says has a reason on the screen.
func KeyReport(exts []Extension) []string {
	var lines []string
	holder := map[string]string{}
	for _, e := range exts {
		for _, c := range e.Claims() {
			switch {
			case !ValidKey(c.Key):
				lines = append(lines, fmt.Sprintf("%s: not a key an extension may claim, asked for by %s", c.Key, e.Name))
			case holder[c.Key] == "":
				holder[c.Key] = e.Name
				lines = append(lines, fmt.Sprintf("%s opens %s, from %s", c.Key, c.Title, e.Name))
			case holder[c.Key] != e.Name:
				lines = append(lines, fmt.Sprintf("%s: %s claims it too, and %s has it, the first by name", c.Key, e.Name, holder[c.Key]))
			}
		}
	}
	return lines
}

// ViewsFile names the file beside an extension's executable that says
// what its views' panes are like, one view per line: the view's id, then
// the settings that differ from a plain list, separated by spaces.
//
//	ask bar=prompt preview=wrap,follow
//
// A view not named there, and every view of an extension without the
// file, is a plain list. A blank line and a line that starts with # are
// skipped, and so is a setting the launcher does not know, so a file
// written for a newer launcher still reads.
const ViewsFile = "views"

// Pane is what a view says about its pane. The zero value is a plain
// list: the bar filters, and the preview is the launcher's own.
type Pane struct {
	// Prompt is bar=prompt: the bar is text to send, not a filter. Typing
	// does not reload the rows, and Enter hands the text to the row under
	// the cursor through the extension's send verb (Send).
	Prompt bool `json:"prompt,omitempty"`
	// Percent is the preview's width, from preview=NN%, in percent of the
	// whole; 0 is the user's own width.
	Percent int `json:"percent,omitempty"`
	// Wrap and Follow are preview=wrap and preview=follow: long lines
	// wrapped, as prose wants, and the end kept in view as the text grows.
	Wrap   bool `json:"wrap,omitempty"`
	Follow bool `json:"follow,omitempty"`
}

// Pane returns what the views file says about the view viewID.
func (e Extension) Pane(viewID string) Pane {
	data, err := os.ReadFile(filepath.Join(e.Dir, ViewsFile))
	if err != nil {
		return Pane{}
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != viewID {
			continue
		}
		return parsePane(fields[1:])
	}
	return Pane{}
}

func parsePane(fields []string) Pane {
	var p Pane
	for _, f := range fields {
		key, value, _ := strings.Cut(f, "=")
		switch key {
		case "bar":
			p.Prompt = value == "prompt"
		case "preview":
			for _, v := range strings.Split(value, ",") {
				switch {
				case v == "wrap":
					p.Wrap = true
				case v == "follow":
					p.Follow = true
				case strings.HasSuffix(v, "%"):
					if n, err := strconv.Atoi(strings.TrimSuffix(v, "%")); err == nil {
						// The same range the user's own width is kept in.
						p.Percent = settings.ClampPreview(n)
					}
				}
			}
		}
	}
	return p
}

// Send runs `<exe> send <id> <text>`: Enter in a pane whose bar is a
// prompt, the text for the row under the cursor. It must return at once,
// so whatever answers is a worker the extension starts and leaves; it is
// stopped after the time a list gets. A non-zero exit means the text was
// not taken, and the caller leaves it in the bar.
func (e Extension) Send(rawID, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	cmd := e.command(ctx, "send", rawID, text)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s: send took longer than %s", e.Name, listTimeout)
	}
	return err
}
