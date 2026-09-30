package apps

import "github.com/beatzball/swoop/internal/protocol"

// Actions are what an app row can do besides open. The ids are what
// swoop-apps run gets as its second argument. Every one of these ends the
// launcher, so all are kind "action"; a "refresh" action would run and
// return to the list instead. RevealTitle is per OS, in the apps_*.go
// files, because the file manager it opens is.
func Actions() []protocol.Item {
	return []protocol.Item{
		{ID: "open", Kind: "action", Icon: "", Title: "Open", Subtitle: "Enter"},
		{ID: "reveal", Kind: "action", Icon: "", Title: RevealTitle, Subtitle: ""},
		{ID: "copy-path", Kind: "action", Icon: "", Title: "Copy path", Subtitle: ""},
	}
}
