# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/). Before 1.0,
a change that breaks the extension contract bumps the middle number and says
so here.

## [Unreleased]

### Fixed

- **`make install` starts the frame the first time.** launchd's bootout
  returns before the old service is gone, and a bootstrap in that window
  was refused, so the frame came up only on a second run. The installer
  now waits for the old service to go and tries again if refused.

### Added

- **Ask AI.** Press Tab from anywhere: a pane with your conversations on
  the left and the transcript on the right. Enter sends what is in the bar;
  dots show under your prompt until the model starts, then the answer
  streams in, and each later prompt appends below. ctrl-k on a
  conversation offers Copy last answer, Copy conversation, and Delete. Esc
  brings the launcher back with the text you had before Tab. The model is
  whatever command you name in `~/.config/swoop/ai`; without that file,
  `claude -p` or `ollama run`. The launchd agents now carry the installing
  shell's PATH, so the frame finds the same tools your terminal does.

## [0.4.1] - 2026-09-25

Two fixes for the frame and for plain terminals, and a new demo.

### Fixed

- **No terminal flash on Enter in the frame.** Enter on an app showed the
  terminal's own screen for a moment before the panel went away. The
  runner now works while fzf still holds the screen, and the frame hides
  before anything else is drawn.
- **Pictures only where they draw.** A terminal that does not speak the
  Kitty graphics protocol showed row icons as boxes and the preview as
  noise. swoop now sends pictures only in a terminal known to draw them
  (Ghostty and the frame, kitty, WezTerm, Konsole); elsewhere rows keep
  their glyphs and the preview starts at the name. `SWOOP_PICTURES=1` or
  `0` overrides the guess.

## [0.4.0] - 2026-09-25

The action menu, a fast first run, and an install with no tools.

### Added

- **The action menu.** ctrl-k on a row, cmd+K in the frame, shows what else
  it can do. Apps offer Open, Reveal in Finder, and Copy path. Clipboard
  History offers Copy and Delete, and Delete removes the entry and brings
  the list back without it. Extensions gain an optional `actions <id>`
  verb, and `run` gets the action's name.
- **Install with no tools.** `curl -fsSL
  https://raw.githubusercontent.com/beatzball/swoop/main/scripts/get | bash`
  installs the latest release, or `--version X.Y.Z` a given one, into
  `~/.local/share/swoop`, with fzf if you have none, and on a Mac starts the
  frame and the clipboard watcher at login. No Go, fzf, or Xcode tools
  needed. Every tag now publishes a release with a tarball for macOS (Apple
  silicon and Intel, frame included), Linux (amd64 and arm64) and Windows.

### Changed

- **The first swoop on a machine opens at once.** It used to spend about
  half a second turning every app icon into a picture before the list
  showed. Now icons not converted yet keep their glyph on that run, and a
  background job converts them, so they are there from the next run.

## [0.3.0] - 2026-09-25

The macOS frame, and swoop as a daily driver.

### Added

- **A macOS frame of our own.** `shell/mac` is a Swift program: a floating
  panel with swoop in it, drawn by libghostty, shown by a global hotkey
  (alt+shift+space by default). The hotkey again, or a click elsewhere,
  hides it with your text kept; Esc ends it, and the next press is a fresh
  swoop, already drawn, because the next one starts while the panel is
  hidden. cmd+plus, cmd+minus and cmd+0 change the font size and the size
  is kept. No Ghostty.app and no permissions needed.
- **A click on a row is Enter on that row**, in every terminal.
- **`make install`.** Builds everything and runs the frame and the clipboard
  watcher at login through launchd, from this checkout. `make uninstall`
  removes them and keeps your history.

## [0.2.0] - 2026-09-25

Clipboard history and file search, both as views. Everything else in
milestone 0.2 shipped in 0.1.0.

### Added

- **Clipboard history.** A watcher, `swoop-clipd`, starts with the launcher
  and keeps what you copy, text only, newest first, in a file of your own
  under the data directory. Enter on "Clipboard History" opens it; typing
  filters; the preview shows the whole entry; Enter copies it back. Never
  kept: a copy carrying the concealed or transient mark, or one made while
  an app on `~/.config/swoop/clipboard.ignore` is in front (password
  managers by default). `swoop-clipd delete` and `clear` remove entries.
- **File search.** Enter on "Search Files" and type part of a name: files
  under your home folder from Spotlight, newest change first, or the files
  you used most recently when nothing is typed. The preview shows a picture
  as a picture, text as text, a folder as its listing, and the facts for the
  rest. Enter opens the file.

## [0.1.0] - 2026-09-24

The first version a person can build and use. macOS only; Linux and Windows
build and test but have no app source or frame yet.

### Added

- **A launcher in a terminal.** `bin/swoop` runs in any terminal on any OS. On
  a Mac, Ghostty's quick terminal is the window: alt+space shows it, Enter
  opens the app, the panel hides on its own (#11, #13).
- **Every app with its real icon on the row** (#18), and a preview pane with
  the large icon, version, bundle id, and path (#16). Pictures travel over the
  Kitty graphics protocol with Unicode placeholders, so fzf can redraw and
  scroll them like text. `swoop-img` shows any picture the same way.
- **Extensions that are programs.** A folder with one executable that answers
  `list`, `preview`, `run`, and `view`, printing tab-separated lines. Found in
  `~/.config/swoop/extensions` and in `SWOOP_EXTENSIONS` (#21, #27).
- **Navigation like Raycast** (#27). Esc clears the bar, then closes. Enter on
  a "view" row opens a pane with its own rows, prompt, and preview; Esc there
  clears, then returns to the root with the old text and the same row
  selected. Enter with no match does nothing (#24).
- **Bundled extensions**: `system` (Sleep, Lock Screen, Toggle Dark Mode,
  Empty Trash) (#21); `define` (Define Word: recent words, alphabetical prefix
  matches from the system word list, definitions from the Mac's own
  dictionary through `swoop-dict`, Enter opens Dictionary.app) (#27); `calc`
  (type `2+2` and a row says `2+2 = 4`; Enter copies the result) (#29).
- **CI on macOS, Linux, and Windows**: gofmt, vet, test, build, shellcheck
  (#14).

### Numbers on the development machine

| what | time |
|---|---|
| startup to first rows | about 30 ms |
| one keystroke at the root | 7 to 9 ms |
| one keystroke inside Define Word | 35 ms |
| the preview of an app | 3 ms warm |

The first run on a machine converts every app icon once and costs about half
a second (#19).

[Unreleased]: https://github.com/beatzball/swoop/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/beatzball/swoop/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/beatzball/swoop/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/beatzball/swoop/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/beatzball/swoop/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/beatzball/swoop/releases/tag/v0.1.0
