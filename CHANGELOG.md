# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/). Before 1.0,
a change that breaks the extension contract bumps the middle number and says
so here.

## [Unreleased]

### Changed

- **The owl in the menu bar.** The frame's menu bar item shows swoop's
  owl as a one-colour mark that takes the bar's colour, light or dark,
  instead of a stock bird symbol.
- **The owl is the logo.** The site's header, hero, touch icon and favicon
  carry it, cut from the drawing with the background removed; the hero
  shows it mirrored and large, larger still on a phone.
- The site at swoop.sh is rebuilt on litro's supernova recipe: a new
  landing page with the install command to copy, the keys worth knowing,
  and what swoop is built on. The docs pages and their addresses are the
  same.

## [0.9.0] - 2026-09-29

Keywords that scope the bar, a docs site, a Snippets pane, and a
launcher that never drops a key.

### Added

- **Keywords**: an extension's keyword and a space ask only that
  extension. `def ap` opens Define with `ap` typed, `win l` shows only the
  window rows for `l`. An extension declares its keyword in a `keyword`
  file beside it; the bundled ones are `def`, `calc`, `clip`, `files`,
  `ai`, `links`, `emoji`, `snip`, `win`, `notes`, `tasks`, `rem`, `stats`
  and `settings`.
- **A docs site, at https://swoop.sh**: install, trying it in a terminal,
  Ask AI, every extension that ships, and Settings, from the README's own
  words. The README links to it at the top.
- **Writing an Extension**, a page of the site that builds one extension
  from nothing: a shell script that lists, previews and runs, then opens a
  pane, then offers ctrl-k actions, then toggles and opens an editor, each
  step tried in the launcher.

### Changed

- **Ask AI loose ends.** With no model picked, ollama runs the model named
  by `ollama = <model>` in the settings file, or else the newest one
  pulled, and the line under the dots says which; `ai = ollama` with no
  model does the same. On Linux the Copy rows say when neither `wl-copy`
  nor `xclip` is installed. On Windows a worker that has stopped is seen
  as stopped, instead of dots until the five minute limit.

- **A New note with nothing typed is named by the date and time**,
  `2026-09-28 21:15` as its heading and `2026-09-28-21-15.md` as its
  file, instead of `untitled`. The New note row says the name it will
  get.
- **New note comes back on the note it made**, with the bar empty, so a
  second Enter does not make a second note. For extensions: a run of a
  `terminal` row may write a row id to the file named by `$SWOOP_LAND`,
  and the launcher comes back on that row with the bar cleared.

### Fixed

- **Snippets: Edit on ctrl-k, and the placeholders explained.** Edit
  opens the snippet's file in your editor. The how-to that the Snippets
  and New snippet rows preview, and the README, say what each of
  `{date}`, `{time}`, `{clipboard}`, `{uuid}` and `{cursor}` becomes.
- **A Snippets row, always.** Once a snippet existed, `snippets` matched
  nothing: each snippet is a row named after itself. Now a Snippets row
  opens a pane, New snippet first, then every snippet, filtered as you
  type; snippets stay at the root too.
- **Snippets with none yet: Enter makes the first one.** The row used to
  post the how-to and close the launcher. Now it writes the Signature
  example and opens it in your editor, in the panel; quit, and it is a
  row. ctrl-k on any snippet has `New snippet`, a file named by the date
  and time, opened the same way.


- **The Settings titles sit in one column.** Editor, Open the config
  folder, the Settings row at the root, and every choice had no icon,
  and the skin tone row's hand was two cells wide, so those titles were
  a cell off from the rest.
- **Keys typed while a list reloads are kept.** Two quick Enters on a
  checklist tick two tasks, and letters typed right after Esc all reach
  the bar. Before, the launcher ignored keys until the new list had
  landed, so the second Enter and the first letters were lost.
- **Accessibility survives a rebuild.** A checkout install makes
  `swoop-dev`, a self-signed code-signing certificate in the login
  keychain (one macOS password prompt, the first time), and `make
  shell-mac` signs the frame with it as `dev.swoop.shell`. Before, every
  build was signed ad-hoc, a new program to macOS, and window rows and
  pastes were refused although the switch still showed on. The permission
  messages now say what to do then: remove swoop-shell-mac with the minus
  button and add it again. Releases are not signed yet, so after an
  upgrade that step is still needed once.

## [0.8.0] - 2026-09-28

What a launcher has: quicklinks, emoji, snippets, window management,
notes, tasks and reminders, all in the panel, with an editor inside it,
a paste that lands where you were typing, and a switch for each.

### Added

- **Edit inside the panel.** Enter on a note, or on `New note`, opens it
  in your editor right in the panel, and quitting brings the list back
  with the preview redrawn; ctrl-k `Open in app` keeps the old way. The
  editor is a new Settings row, `editor = nvim` in the config, else
  `$EDITOR`, else `nano`. Tasks gets ctrl-k `Edit the list`, Search Files
  gets ctrl-k `Edit` on a text file. Extensions can use it too: a row of
  kind `terminal` gets the whole terminal and the launcher stays open.
- **Notes.** Type `notes`, Enter: markdown files in
  `~/.local/share/swoop/notes/`, newest change first, searched by title
  and text, previewed rendered. Enter opens the file; `New note` makes
  one from what you typed; ctrl-k copies, reveals, or deletes to
  `deleted/`.
- **Tasks.** Type `tasks`, Enter: a checklist in
  `~/.local/share/swoop/tasks.md`. Typed text adds a task, and
  `today`, `tomorrow`, a weekday or a date at its end is the due date.
  Enter ticks a task and the list stays open; ctrl-k undoes, deletes,
  copies. On a Mac, `reminders` is the same view over Apple Reminders.
- **Snippets.** Named text as rows at the root, one markdown file each in
  `~/.config/swoop/snippets/`: name, an optional `keyword:` line, text.
  Enter pastes into the app in front with `{date}`, `{time}`,
  `{clipboard}` and `{uuid}` filled, and leaves the caret at `{cursor}`;
  it copies where emoji copies. `swoop-snippets import` reads a JSON
  export (name, text, keyword); ctrl-k copies or deletes.
- **Emoji and symbols.** Type `emoji`, Enter, then a name or keyword:
  every emoji and flag, and a few hundred symbols. Enter pastes into the
  app in front, or copies without Accessibility, in a plain terminal,
  and on Linux; ctrl-k copies or picks one of six skin tones, and the
  default tone is a setting. The most recently used lead the view.
- **Window management.** Rows that move and size the window you were in:
  halves, thirds, quarters, maximize, almost maximize, reasonable size,
  center, next and previous display. macOS, through the Accessibility
  API; the first run without the permission opens the settings pane.
- **Quicklinks.** Named links as rows at the root, from
  `~/.config/swoop/quicklinks.tsv`: name, link, app to open it with. A
  link holding `{argument}` is a pane: Enter, type, Enter opens the link
  with the text in it, encoded. Google, DuckDuckGo, Wikipedia, YouTube,
  and GitHub ship until the file exists. `swoop-links add` and
  `swoop-links import` (a JSON export) write the file; ctrl-k copies
  the link or deletes it.
- **Apps on Linux.** swoop lists the applications in `.desktop` files, in
  XDG order, so a copy in `~/.local/share/applications` overrides or hides
  the system's. Enter launches with `gio launch`, or runs the Exec line
  when gio is missing; the menu shows the folder and copies the path. The
  preview shows the name, the command, the comment, and the file.
- **Start, stop, restart.** `swoop start|stop|restart|status` works the
  same from Homebrew, the curl installer, or a checkout; the bird menu
  has Restart and Quit; and after an upgrade the frame restarts itself
  once the panel is hidden.
- **Turn extensions on and off.** Settings, Extensions lists every
  extension, bundled and your own; Enter turns one off, and it leaves the
  root, Tab and ctrl-k until you turn it back on. It is one line in the
  config, `off = reminders, tasks`, which you can also write by hand.
  Settings itself stays on.

### Changed

- **Emoji and snippets paste from the frame.** The frame hides its panel,
  then presses cmd+V itself, and only into a text field (or an app that
  shows no focus, like a terminal); on a button or the desktop the text
  is copied and a notification says so. No more osascript keystroke or
  fixed wait, and the Accessibility prompt now names swoop-shell-mac.
- **Lighter tools.** The twelve tools together are 30 MB stripped, from
  35. swoop-settings no longer carries an HTTP stack for one request to
  LM Studio on localhost; the Kitty placeholder table and a small width
  function replace a Charm package that brought four modules along for
  two constants and one measurement.

### Fixed

- **The Reminders pane is no longer empty.** It opens at once, from a
  cache or with a `Loading reminders…` row, and fills itself when the
  list arrives. The list is read through EventKit in a fraction of a
  second, not in 40 through Apple Events.

## [0.7.0] - 2026-09-27

Settings inside the launcher, Ask AI with any model, and the list that
learns what you use.

### Added

- **Used recently, and Stats.** The root list starts with the five things
  opened most recently, marked. Every open is one line in
  `~/.local/state/swoop/usage.jsonl`; a `Stats` row shows what you open
  most, how often, when last, and a fortnight by day. A setting turns the
  group off; ctrl-k in Stats clears the log.
- **Start Screensaver**, a row beside Sleep and Lock Screen.
- **ctrl-k on New conversation changes the model.** The same list as
  Settings, Enter picks one and returns to the pane; the New row says
  which model will answer.
- **Ask AI with any model.** Presets under AI model in Settings: `claude`,
  `codex`, each `ollama` model, each model LM Studio has loaded, and
  `openai:<model>` for OpenAI or any server with that API (OpenRouter,
  Groq, vLLM) with the address under API URL and the key under API key.
  Two transports: a command with the conversation on stdin, or a small
  HTTP client in the OpenAI chat shape, streamed, with a model's thinking
  shown as "thinking" under the dots rather than in the answer. Web
  search, when on, reaches claude, codex and OpenAI; the line under the
  dots says when the model you picked cannot search.
- **Settings, inside the launcher.** Type `settings`, or cmd+, in the
  frame: one row per setting, Enter to change it. The hotkey, the preview
  width, the AI model (claude, or ollama with any model you have), web
  search on or off, the transcript renderer, and a door to the config
  folder. Every value is one line in `~/.config/swoop/config`. The frame
  watches that file, so a new hotkey works within a second.
- **A Homebrew tap.**

### Changed

- **Web search is off by default.** claude may search and fetch only when
  the Web search setting is on. Turn it on in Settings, or `web = on` in
  the config file. `brew install beatzball/tap/swoop`, and on a Mac
  `brew services start swoop` runs the frame at login. The formula is
  written by `scripts/tap-formula` from a release's checksums, and the
  release workflow pushes it to the tap when it has a token for it.

### Fixed

- **ollama answers are clean.** `ollama run` wrapped words with cursor
  moves even through a pipe, and a thinking model's reasoning came out
  as the answer. The ollama line now runs with `--nowordwrap`,
  `--hidethinking` and `--think=false`, escape codes are stripped from
  what any command prints, and with web search on and an ollama model
  the line under the dots says the model cannot search.

## [0.6.1] - 2026-09-26

The renderer as a command, and the watcher put to rest.

### Fixed

- **The launcher never starts a watcher launchd owns.** On a Mac with
  the agent installed, `swoop-clipd start` now does nothing: launchd
  brings its watcher back itself, and one started by the launcher in the
  second before it did held the lock from then on. The installer also
  waits for launchd's watcher to hold the lock before it starts the
  frame.

### Added

- **`swoop-md`.** The transcript's markdown renderer as a command:
  markdown on stdin, styled text on stdout, `-w` for the width. For
  anything else that wants markdown in a terminal.
- **`render =` in the settings file.** Any command can draw the AI
  transcript instead of the built-in renderer: `render = glow -s dark`.
  The answer goes to its stdin, its stdout is the pane.

## [0.6.0] - 2026-09-26

The frame grows up: a movable divider, a resizable panel, a bird in the
menu bar. And tests for the wiring.

### Added

- **The divider moves.** cmd+[ and cmd+] in the frame, alt+left and
  alt+right in a terminal, give the preview or the list 5% more, and the
  width is remembered in `~/.config/swoop/config`, the launcher's own
  settings file: one `key = value` per line.
- **The panel resizes.** Drag any edge of the frame; the size is kept in
  `shell-mac.json` beside the font size, and can be typed there too.
- **A handle on the edge.** Bring the mouse near an edge of the panel
  and a short bar appears there, a small mark near a corner: the edges
  drag. Nothing shows anywhere else.
- **A bird in the menu bar.** Open swoop, Settings, Quit. The one visible
  sign the frame is running.
- **Two end-to-end tests.** `scripts/launcher-test` drives the launcher
  through a pseudo-terminal: filter, a view and back, the action menu,
  Ask AI with a fake model, the divider keys, Enter on a row.
  `scripts/launchd-test` runs the installer's launchd steps against a
  fake launchctl and checks the order that keeps a stray watcher from
  winning. `make e2e` and `make test`; CI runs both.

### Changed

- **swoop-ai is half the size.** The AI transcript is rendered by
  swoop's own markdown renderer on goldmark, in place of glamour, which
  brought a code highlighter for every language along: 17.6 MB to
  9.3 MB, and 6.4 MB in the release tarball, which is now built with
  symbols stripped, as every tool in it is.

### Fixed

- **The launcher opens on Linux.** The app lister answered "not
  implemented" as an error there, and `bin/swoop` stopped before fzf
  started. It now answers with no apps, and the extensions' rows show.
  Found by the new launcher test on its first Linux run.
- **`make install` unloads before it kills.** A frame killed while its
  launchd service was still loaded came back within a second, ran swoop,
  and started a clipboard watcher outside launchd, which launchd's own
  then gave way to. The installer now unloads both services, waits for
  launchd to forget them, then stops what is left, then starts.

## [0.5.0] - 2026-09-25

Ask AI: a conversation pane on Tab, fed by whatever command you name.

### Fixed

- **`make install` leaves one clipboard watcher, launchd's.** A watcher
  still letting go of its lock made launchd's new one give up, and the
  frame's own swoop then started one outside launchd that won forever.
  The installer now waits for the old frame and watcher to be gone before
  it starts anything.
- **`make install` starts the frame the first time.** launchd's bootout
  returns before the old service is gone, and a bootstrap in that window
  was refused, so the frame came up only on a second run. The installer
  now waits for the old service to go and tries again if refused.

### Fixed

- **An answer that never comes says so.** A command that ended with
  nothing on stdout looked like one still thinking. It is now a red line
  under the prompt with the reason: the command's last words on stderr,
  or that it printed nothing. While waiting, the dots carry the seconds
  and the last thing the command said, which for the default `claude -p`
  is the tool it is using. A worker that dies is reported, not waited
  for; five minutes is the most an answer may take.

### Changed

- **App rows show only the name.** The folder is in the preview.
- **The preview is wider**, 58% of the window, and ctrl-u and ctrl-d
  scroll it half a page.
- **The AI transcript is rendered.** Answers are markdown, drawn by
  glamour: headings, lists, code, tables. The default `claude -p` may now
  search and fetch the web, so a question about the weather gets an
  answer instead of a refusal.

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
- **Navigation like the launchers you know** (#27). Esc clears the bar, then closes. Enter on
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

[Unreleased]: https://github.com/beatzball/swoop/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/beatzball/swoop/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/beatzball/swoop/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/beatzball/swoop/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/beatzball/swoop/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/beatzball/swoop/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/beatzball/swoop/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/beatzball/swoop/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/beatzball/swoop/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/beatzball/swoop/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/beatzball/swoop/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/beatzball/swoop/releases/tag/v0.1.0
