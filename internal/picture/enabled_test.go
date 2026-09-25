package picture

import "testing"

func TestEnabled(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"nothing known", map[string]string{"TERM": "xterm-256color"}, false},
		{"ghostty", map[string]string{"TERM_PROGRAM": "ghostty"}, true},
		{"wezterm", map[string]string{"TERM_PROGRAM": "WezTerm"}, true},
		{"kitty by window id", map[string]string{"KITTY_WINDOW_ID": "1"}, true},
		{"kitty by TERM", map[string]string{"TERM": "xterm-kitty"}, true},
		{"ghostty by TERM only", map[string]string{"TERM": "xterm-ghostty"}, true},
		{"konsole", map[string]string{"KONSOLE_VERSION": "230800"}, true},
		{"apple terminal", map[string]string{"TERM_PROGRAM": "Apple_Terminal", "TERM": "xterm-256color"}, false},
		{"tmux under ghostty", map[string]string{"TERM_PROGRAM": "tmux", "TERM": "tmux-256color"}, false},
		{"forced on", map[string]string{"SWOOP_PICTURES": "1", "TERM": "dumb"}, true},
		{"forced off", map[string]string{"SWOOP_PICTURES": "0", "TERM_PROGRAM": "ghostty"}, false},
		{"junk setting falls through", map[string]string{"SWOOP_PICTURES": "yes", "TERM_PROGRAM": "ghostty"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := enabled(func(k string) string { return c.env[k] })
			if got != c.want {
				t.Errorf("enabled(%v) = %v, want %v", c.env, got, c.want)
			}
		})
	}
}
