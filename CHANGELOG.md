# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/). Before 1.0,
a change that breaks the extension contract bumps the middle number and says
so here.

## [Unreleased]

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

[Unreleased]: https://github.com/beatzball/swoop/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/beatzball/swoop/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/beatzball/swoop/releases/tag/v0.1.0
