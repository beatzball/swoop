package main

import (
	"bytes"
	"strings"
	"testing"
)

// Rows in, the matching ones out, best first, a header only over rows
// that match; a line that is not a row is dropped.
func TestRun(t *testing.T) {
	in := strings.Join([]string{
		"1\ttoggle\t\tFeed the monkey\t",
		"not a row",
		"2\ttoggle\t\tAPI key\tnot set",
		"g\tgroup\t\tLater\t2 rows",
		"3\ttoggle\t\tNothing\t",
		"",
	}, "\n")
	for query, want := range map[string]string{
		"key":  "2 1",
		"akey": "2",
		"set":  "2",
		"noth": "g 3",
		"":     "1 2 g 3",
		"zzz":  "",
	} {
		var out bytes.Buffer
		if err := run(query, strings.NewReader(in), &out); err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
			if id, _, _ := strings.Cut(line, "\t"); id != "" {
				ids = append(ids, id)
			}
		}
		if got := strings.Join(ids, " "); got != want {
			t.Errorf("%q: got %q, want %q", query, got, want)
		}
	}
}
