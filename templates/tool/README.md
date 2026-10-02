# @NAME@

A keyboard tool built on [swoopkit](https://swoop.sh/docs/build-your-own):
`fzf` does the finding, and every row comes from a small program in
`extensions/` that prints lines.

```
tool                     the name, title, id and hotkey
bin/@NAME@               the command: it names the tool file and runs the kit's launcher
extensions/hello/hello   one extension: three rows, a preview, a run
```

## Run it

It needs swoopkit's launcher, `swoop`, and `fzf` on your `PATH`.

```sh
bin/@NAME@          # opens in this terminal; Esc quits
bin/@NAME@ tool     # what the tool file says, filled in
```

Type to filter, Enter runs the row. The rows are the ones `extensions/`
prints and no others.

To have it as a command, link it into a folder on your `PATH`:

```sh
ln -s "$PWD/bin/@NAME@" ~/.local/bin/@NAME@
```

## Make it yours

- **Add a row.** Edit `extensions/hello/hello`, or add a folder beside it
  with one program in it, named the same. `list` prints one row a line,
  five fields with tabs between: id, kind, icon, title, subtitle.
  `preview <id>` prints the right-hand pane and `run <id>` does the thing.
  [Writing an Extension](https://swoop.sh/docs/writing-an-extension) builds
  one a step at a time
- **Name it.** `tool` holds the title shown in the frame's menu, the id of
  its launchd labels, and the key that opens it
- **A person's own extensions** go in `~/.config/@NAME@/extensions`, and
  join the ones here

## The frame, on a Mac

The frame is the panel that opens on the hotkey. It needs swoopkit's
frame, which a release of swoop for the Mac has.

```sh
bin/@NAME@ start      # install it: now, and at every login
bin/@NAME@ status     # is it running, where from, on which key
bin/@NAME@ restart    # install it again: after a new version of swoopkit
bin/@NAME@ stop       # until the next login, or the next start
```

`start` gives this tool a frame of its own: a copy of swoopkit's in
`~/.local/share/@NAME@/frame`, signed under the `id` in `tool`, and two
launchd agents, `<id>.shell` and `<id>.clipd`, that run it and the
clipboard watcher. It runs beside swoop's frame, or any other tool's:
its own process, its own key, its own folders, and its own entry under
Accessibility in System Settings. Two tools on one Mac need two keys;
the `hotkey` in `tool` is this one's.

To take it out for good: `bin/@NAME@ stop`, then delete the two files
`~/Library/LaunchAgents/<id>.shell.plist` and `<id>.clipd.plist`.

To try the frame with nothing installed, start swoopkit's with
`SWOOP_LAUNCHER` naming this tool's command:

```sh
SWOOP_LAUNCHER="$PWD/bin/@NAME@" swoop-shell-mac
```

[Build your own tool](https://swoop.sh/docs/build-your-own) has the rest:
the folders, the frame, and how to ship it.
