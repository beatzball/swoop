---
title: Getting Started
description: Install swoop on a Mac or on Linux, press the hotkey, or run the same launcher in any terminal.
sidebar:
  order: 1
---

## What it is

swoop is an open-source keyboard launcher built the Unix way. fzf does the
finding, libghostty does the drawing, and every extension is a program that
prints lines.

![swoop: filtering apps, the action menu, the calculator, and Ask AI answering a question](/swoop.webp)

Press the hotkey, type a few letters, press Enter. Apps, files, quicklinks,
emoji, snippets, notes, tasks, window moves and an AI you choose all live in
one list. ctrl-k on any row opens what else it can do.

## Install

With Homebrew, on a Mac or on Linux:

```sh
brew install beatzball/tap/swoop
brew services start swoop   # Mac: the frame at login, alt+shift+space
```

Or with nothing but curl:

```sh
curl -fsSL https://raw.githubusercontent.com/beatzball/swoop/main/scripts/get | bash
```

No Go, no Xcode tools, no fzf needed. It downloads the release for your OS
and arch into `~/.local/share/swoop/<version>`, links
`~/.local/share/swoop/current` to it, and fetches fzf into the same place if
fzf is not on your PATH. On a Mac it also writes two launchd agents that run
at login: the frame, which owns the hotkey and the panel, and the clipboard
watcher. Press alt+shift+space. On Linux and Windows there is no frame yet;
it tells you how to run `swoop` in a terminal.

`| bash -s -- --version 0.4.0` installs a given release instead of the
latest. Run it again to upgrade. Logs are in `~/Library/Logs/swoop`, and
`~/.local/share/swoop/current/scripts/uninstall` removes the agents and keeps
your history.

## Start, stop, status

However it was installed:

| command | what it does |
|---|---|
| `swoop restart` | restarts the frame and the clipboard watcher |
| `swoop stop` | stops them |
| `swoop start` | starts them |
| `swoop status` | says whether they run, where from, and how they were installed, which extension has which key, and which lists were too slow the last time the launcher was slow to open |

The bird in the menu bar has the same: Restart, and Quit, which keeps it
down until the next login or `swoop start`. After an upgrade the frame
restarts itself, once the panel is hidden.

Releases are not signed yet, so macOS takes each upgraded frame for a new
program: after `brew upgrade` or a new release, remove swoop-shell-mac from
System Settings, Privacy & Security, Accessibility with the minus button and
add it again. The switch shows on either way; only a new grant works.

## From a checkout

```sh
git clone https://github.com/beatzball/swoop.git && cd swoop && make install
```

The same two agents, pointed at the checkout. You need Go, fzf, and Xcode's
tools. `make install` again after a pull restarts them on the new build;
`make uninstall` removes them and keeps your history.

The first `make install` also makes `swoop-dev`, a code-signing certificate
in your login keychain, and signs the frame with it, so macOS sees every
build as the same program and an Accessibility grant survives the next
install. Trusting it shows one macOS password prompt, the first time only.

## Try it in any terminal

Without the frame, swoop is still a terminal program: the same one the frame
runs. You need Go and fzf. Then, in a checkout:

```sh
make build        # compiles the tools into bin/
bin/swoop         # lists your apps; type to filter; Enter opens; Esc quits
```

On Linux the apps come from `.desktop` files, in XDG order: your own
`~/.local/share/applications` first, then each of `XDG_DATA_DIRS`.

It runs in any terminal. Icons and previews are pictures in a terminal that
draws them (Ghostty, kitty, WezTerm, Konsole) and glyphs and text anywhere
else; `SWOOP_PICTURES=1` or `0` overrides the guess. This is also how you
run it on Linux or Windows.

### In Ghostty's quick terminal

If you prefer Ghostty's quick terminal to the frame, add to your Ghostty
config:

```
keybind = global:alt+space=toggle_quick_terminal
quick-terminal-position = center
```

Reload it with `cmd+shift+,`, grant Ghostty Accessibility access when it
asks, and add to the end of your `~/.zshrc`, with the path to your checkout:

```sh
# Ghostty sets this in its quick terminal and nowhere else.
if [[ -n "$GHOSTTY_QUICK_TERMINAL" ]] && [[ -x /absolute/path/to/swoop/bin/swoop ]]; then
  exec /absolute/path/to/swoop/bin/swoop
fi
```

Take those lines out again to get a plain quick terminal back.

## The macOS frame

The frame is a small Swift program, `shell/mac`, that does nothing but show
a floating panel with swoop in it, drawn by libghostty, on a global hotkey.
`make install` runs it for you; to run it by hand instead:

```sh
make build shell-mac
SWOOP_LAUNCHER="$PWD/bin/swoop" shell/mac/.build/release/swoop-shell-mac
```

Then press alt+shift+space. Esc closes it, Enter opens what you picked, a
click elsewhere hides it, and the next press is a fresh launcher with an
empty bar. `SWOOP_HOTKEY=alt+space` picks another key. `SWOOP_LAUNCHER`
names the launcher the frame runs, and it is how `swoop status` and a
later `make install` know a frame started by hand as swoop's. It needs no
Ghostty.app. One permission, Accessibility, and only to paste emoji and
snippets for you: macOS asks the first time.

| key | what it does |
|---|---|
| `cmd+[` / `cmd+]` | move the divider between the list and the preview, 5% a step (alt+left and alt+right in a plain terminal) |
| `cmd+,` | open [Settings](/docs/settings) (alt+, in a plain terminal) |
| `ctrl-k` | the actions for the row under the cursor |
| `Tab` | [Ask AI](/docs/ask-ai), with what you typed |

The divider's place is remembered in `~/.config/swoop/config`
(`preview = 58`). Drag any edge of the panel to resize it; the size is
remembered in `~/.config/swoop/shell-mac.json` beside the font size, and can
be typed there too (`width`, `height`). A bird in the menu bar opens the
launcher, opens it on the Settings pane, and quits; `SWOOP_NO_MENU_BAR=1` leaves
it out.

## What runs when the launcher is closed

One thing: `swoop-clipd`, the clipboard watcher. See
[Clipboard History](/docs/extensions#clipboard-history) for what it keeps and
what it never keeps.

## Next

- [Extensions](/docs/extensions): everything that ships in the list
- [Ask AI](/docs/ask-ai): Tab, and any model you like
- [Writing an Extension](/docs/writing-an-extension): add your own rows with
  a shell script
