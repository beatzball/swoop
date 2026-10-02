package ext

import (
	"os"
	"path/filepath"
	"strings"
)

// CacheFile names the file beside an extension's executable that says how
// long its root rows keep: the word "run" on the first line means they
// are listed once per launch, with no text, and kept until the launcher
// closes. That is for a list that is long, does not depend on the bar,
// and does not change while the launcher is open: the applications on
// the machine. Without it the root asks `list` again on every keystroke,
// with the text. A file, as the keyword is, so the launcher knows before
// it runs anything.
const CacheFile = "cache"

// IconsFile names the file beside an extension's executable that asks for
// pictures on its rows: the word "id" on the first line means each row's
// id is the path of a file, and the icon the row shows is that file's
// own, in place of the glyph. swoop-icons draws them and warms its cache
// in the background. It counts only together with CacheFile: a terminal
// takes the pictures once, as the launcher starts, so the rows must be
// known by then.
const IconsFile = "icons"

// word is the first word of the first line of a file beside the
// extension, or "" without one.
func (e Extension) word(file string) string {
	data, err := os.ReadFile(filepath.Join(e.Dir, file))
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Once says whether the extension's root rows are listed once per launch
// and kept for the run. See CacheFile.
func (e Extension) Once() bool {
	return e.word(CacheFile) == "run"
}

// IconsByID says whether the extension's rows take their icon from the
// file their id names. See IconsFile.
func (e Extension) IconsByID() bool {
	return e.Once() && e.word(IconsFile) == "id"
}
