# swoop

Say it like a bird does: swoop down, grab the thing, gone.

An open-source keyboard launcher built the Unix way: fzf does
the finding, libghostty does the drawing, and every extension is a program
that prints lines.

Early days. There is nothing to install yet, but there is something to try.

## Try it in any terminal

Without the frame, swoop is still a terminal program. You need Go and fzf.
Then:

```sh
make build        # compiles the tools into bin/
bin/swoop         # lists your apps; type to filter; Enter opens; Esc quits
```

It runs in any terminal. To make it feel like a launcher, let Ghostty's quick
terminal run it. Add to your Ghostty config:

```
keybind = global:alt+space=toggle_quick_terminal
quick-terminal-position = center
```

and to the end of your `~/.zshrc`, with the path to your checkout:

```sh
# Ghostty sets this in its quick terminal and nowhere else.
if [[ -n "$GHOSTTY_QUICK_TERMINAL" ]] && [[ -x /absolute/path/to/swoop/bin/swoop ]]; then
  exec /absolute/path/to/swoop/bin/swoop
fi
```

Two more steps, or the hotkey does nothing:

1. Ghostty does not reload its config when the file changes. Press
   `cmd+shift+,` in any Ghostty window, or quit and reopen it.
2. A `global:` keybind needs macOS Accessibility access. Ghostty asks for it
   on that reload. If no dialog appears, open System Settings, then Privacy &
   Security, then Accessibility, and switch Ghostty on.

Now alt+space shows swoop. Enter opens the app, swoop exits, and the panel
closes on its own. Next alt+space starts fresh.

If alt+space is already taken on your Mac, pick another key, such as
`global:ctrl+alt+space=toggle_quick_terminal`.

![swoop listing apps and filtering as you type](demo/spike.gif)

The plan, every design decision, and the work in progress live in the
[issues](https://github.com/beatzball/swoop/issues). Start there.

## Install

```sh
git clone https://github.com/beatzball/swoop.git && cd swoop && make install
```

That builds everything and writes two launchd agents that run at login: the
frame, which owns the hotkey and the panel, and the clipboard watcher. Press
alt+shift+space. `make install` again after a pull restarts them on the new
build; `make uninstall` removes them and keeps your history. Logs are in
`~/Library/Logs/swoop`. You need Go, fzf, and Xcode's tools; a build with no
tools needed is the next step.

## The macOS frame

The frame is a small Swift program, `shell/mac`, that does nothing but show
a floating panel with swoop in it, drawn by libghostty, on a global hotkey.
`make install` runs it for you; to run it by hand instead:

```sh
make build shell-mac
PATH="$PWD/bin:$PATH" shell/mac/.build/release/swoop-shell-mac
```

Then press alt+shift+space. Esc closes it, Enter opens what you picked, a
click elsewhere hides it, and the next press is a fresh launcher with an
empty bar. `SWOOP_HOTKEY=alt+space` picks another key; `SWOOP_LAUNCHER`
points at a different `swoop`. It needs no Ghostty.app and no permission.

## What runs when the launcher is closed

One thing: `swoop-clipd`, the clipboard watcher. The launcher starts it the
first time and it keeps running, polling the clipboard a few times a second
and appending new text to `~/.local/share/swoop/clipboard/history.jsonl`,
readable by you only. Two things it never keeps: a copy that carries the
"concealed" mark some password managers set, and a copy made while an app on
the ignore list is in front. The list defaults to the known password managers;
write your own, one bundle id per line, at `~/.config/swoop/clipboard.ignore`.
Copy a password from your manager and run `bin/swoop-clipd types` to see what
it marks, and `osascript -e 'id of app "Its Name"'` to get its bundle id.

If a manager still gets through, start the watcher with `SWOOP_CLIPD_DEBUG=1`
and read `watcher.log` next to the history: each change is logged with the app
in front and the marks seen, never the text.

`swoop-clipd status` says whether it is running; `swoop-clipd delete <id>` and
`swoop-clipd clear` remove entries; kill it if you would rather it did not run.

## Write an extension

An extension is a folder with one program in it, named the same:

```
~/.config/swoop/extensions/hello/hello
```

The program answers four commands. Each is one run, then exit:

```
hello list                 # print one line per result
hello preview <id>         # print the right-hand pane for one result
hello run <id>             # do it
hello view <id> [query]    # print the rows of a pane, for a row of kind "view"
```

A row of kind `view` opens a pane instead of running: the launcher asks
`view` for its rows, again on every keystroke, and shows exactly what comes
back. `extensions/define/define` is one: Enter on "Define Word" and you are
typing into the dictionary.

A result line is five fields separated by tabs. Only the last three are shown:

```
id	kind	icon	title	subtitle
```

That is the whole contract. A shell script is enough; `extensions/system/system`
in this repository is one, and it is what puts Sleep and Lock Screen in the
list. Any language works, as long as it starts fast: `list` runs when the
launcher opens and `preview` on every cursor move.

## License

[MIT](LICENSE)
