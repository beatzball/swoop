# swoop

Say it like a bird does: swoop down, grab the thing, gone.

An open-source keyboard launcher built the Unix way: fzf does
the finding, libghostty does the drawing, and every extension is a program
that prints lines.

Early days. There is nothing to install yet, but there is something to try.

## Try it

You need Go and fzf. Then:

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
`global:cmd+shift+space=toggle_quick_terminal`.

![swoop listing apps and filtering as you type](demo/spike.gif)

The plan, every design decision, and the work in progress live in the
[issues](https://github.com/beatzball/swoop/issues). Start there.

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
