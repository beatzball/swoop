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

Now alt+space shows swoop. Enter opens the app, swoop exits, and the panel
closes on its own. Next alt+space starts fresh.

![swoop listing apps and filtering as you type](demo/spike.gif)

The plan, every design decision, and the work in progress live in the
[issues](https://github.com/beatzball/swoop/issues). Start there.

## License

[MIT](LICENSE)
