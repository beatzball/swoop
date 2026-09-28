package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDesktop(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseDesktop(t *testing.T) {
	d, err := ParseDesktop(strings.NewReader(`# a comment
[Desktop Entry]
Type=Application
Name = Text Editor
Name[de]=Texteditor
Comment=Edit\stext files
Exec=gnome-text-editor %U
Icon=org.gnome.TextEditor

[Desktop Action new-window]
Name=New Window
Exec=gnome-text-editor --new-window
`))
	if err != nil {
		t.Fatal(err)
	}
	want := Desktop{
		Type:    "Application",
		Name:    "Text Editor",
		Exec:    "gnome-text-editor %U",
		Icon:    "org.gnome.TextEditor",
		Comment: "Edit text files",
	}
	if d != want {
		t.Fatalf("got %+v, want %+v", d, want)
	}
	if !d.Listed() {
		t.Fatal("a plain application should be listed")
	}
}

func TestStripFieldCodes(t *testing.T) {
	for in, want := range map[string]string{
		"firefox %u":                 "firefox",
		"code --new-window %F":       "code --new-window",
		"app %i %c %k --flag":        "app    --flag",
		"printf 100%%":               "printf 100%",
		`sh -c "echo hi"`:            `sh -c "echo hi"`,
		"env FOO=1 app --file=%f -x": "env FOO=1 app --file= -x",
		"odd %z":                     "odd %z",
	} {
		if got := StripFieldCodes(in); got != want {
			t.Errorf("StripFieldCodes(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDesktopItems(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user", "applications")
	system := filepath.Join(root, "system", "applications")

	normal := writeDesktop(t, system, "editor.desktop",
		"[Desktop Entry]\nType=Application\nName=Editor\nExec=editor %F\n")
	writeDesktop(t, system, "handler.desktop",
		"[Desktop Entry]\nType=Application\nName=Handler\nExec=h\nNoDisplay=true\n")
	writeDesktop(t, system, "gone.desktop",
		"[Desktop Entry]\nType=Application\nName=Gone\nExec=g\nHidden=true\n")
	writeDesktop(t, system, "site.desktop",
		"[Desktop Entry]\nType=Link\nName=Site\nURL=https://example.com\n")
	writeDesktop(t, system, "readme.txt", "not a desktop file")

	// The same file name in both: the user's copy wins, both when it
	// renames the app and when it hides one the system ships.
	writeDesktop(t, system, "term.desktop",
		"[Desktop Entry]\nType=Application\nName=System Terminal\nExec=term\n")
	userTerm := writeDesktop(t, user, "term.desktop",
		"[Desktop Entry]\nType=Application\nName=My Terminal\nExec=term --mine\n")
	writeDesktop(t, system, "ads.desktop",
		"[Desktop Entry]\nType=Application\nName=Ads\nExec=ads\n")
	writeDesktop(t, user, "ads.desktop",
		"[Desktop Entry]\nType=Application\nName=Ads\nHidden=true\n")

	missing := filepath.Join(root, "nowhere", "applications")
	items := desktopItems([]string{user, missing, system})

	got := map[string]string{}
	for _, it := range items {
		got[it.Title] = it.ID
		if it.Kind != Kind || it.Icon != Icon || it.Subtitle != "" {
			t.Errorf("row %q: kind %q icon %q subtitle %q", it.Title, it.Kind, it.Icon, it.Subtitle)
		}
	}
	want := map[string]string{"Editor": normal, "My Terminal": userTerm}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for title, id := range want {
		if got[title] != id {
			t.Errorf("%q: id %q, want %q", title, got[title], id)
		}
	}
}
