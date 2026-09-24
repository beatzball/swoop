package clip

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// IgnorePath is the list of apps whose copies are never kept: one bundle
// id per line, blank lines and # comments allowed. Password managers go
// here. The pasteboard does not say which app wrote to it, so the watcher
// matches the app that was in front at that moment, which is the same
// guess Raycast makes.
func IgnorePath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "swoop", "clipboard.ignore")
}

// DefaultIgnore is what the watcher uses when no list exists: the password
// managers whose bundle ids are known. A user's own list replaces it.
var DefaultIgnore = []string{
	"com.1password.1password",
	"com.agilebits.onepassword7",
	"com.bitwarden.desktop",
	"com.apple.Passwords",
	"com.lastpass.LastPass",
	"com.dashlane.dashlanephonefinal",
	"me.proton.pass.electron",
	"com.keepassxc.keepassxc",
}

// LoadIgnore reads the list at path, or returns DefaultIgnore when there
// is none. A list that exists but is empty means "ignore nothing".
func LoadIgnore(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return set(DefaultIgnore), nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ids []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ids = append(ids, line)
	}
	return set(ids), sc.Err()
}

func set(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[strings.ToLower(id)] = true
	}
	return m
}
