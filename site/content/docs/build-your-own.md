---
title: Build Your Own Tool
description: Make a keyboard tool of your own on swoopkit, the toolkit under swoop, in one command. What the folder holds, how to add rows, how to name the frame, and how to ship it.
sidebar:
  order: 6
---

## What swoopkit is

swoopkit is the toolkit swoop is built on: a launcher script that wires a
few small programs into one `fzf` call, a contract for the programs that
print the rows, and a frame that shows it all in a panel on a hotkey. The
kit has no rows of its own and no name to show. A tool is a folder with
an `extensions` folder and, beside it, one file called `tool` that says
what the tool is. swoop is that file plus its bundled extensions. A
picker for your team's runbooks, or a launcher for one product, is
another file and your own extensions, with the kit's programs unchanged.

## Make one

You need swoop installed: its command, `swoop`, is the kit's launcher.
[Getting Started](/docs/getting-started) has the ways. Then, in any
folder:

```sh
swoop new mytool
mytool/bin/mytool
```

The first line makes a folder called `mytool`. The second opens it in
your terminal: three rows, Hello one, Hello two and Hello three, and none
of swoop's. Type to filter, Enter runs the row, Esc quits.

`new` takes one plain word: letters, digits, and `.` `_` `-` `+` after
the first. It writes over nothing: if `mytool` is already there, it says
so and stops.

From a checkout of the repository the launcher is `bin/swoop`, and it is
not on your `PATH`. `new` says so, and prints the line that runs the tool
anyway:

```sh
make build
bin/swoop new mytool
PATH="$PWD/bin:$PATH" mytool/bin/mytool
```

## What the folder holds

```
mytool/
  tool                     the name, title, id and hotkey
  bin/mytool               the command
  extensions/hello/hello   one extension: three rows, a preview, a run
  README.md                how to run it and change it
```

`tool` is a key and a value on each line:

```
name mytool
title mytool
id dev.mytool
hotkey ctrl+alt+space
```

| key | what it names |
|---|---|
| `name` | the tool's folders, `~/.config/mytool`, `~/.local/share/mytool` and `~/.local/state/mytool`, and its command. One plain word |
| `title` | the name as shown: the frame's menu, a notification. Spaces are fine |
| `id` | the prefix of the launchd labels on a Mac, `<id>.shell` and `<id>.clipd`, and of the frame's signature. A domain of yours, backwards |
| `hotkey` | the key that opens the frame, until the person picks one. `new` gives it a key that is not swoop's, so the two run side by side |

A key left out is filled in: the title is the name, the id is
`dev.<name>`. `mytool/bin/mytool tool` prints the four lines as the kit
reads them, and `mytool/bin/mytool tool name` one value.

`bin/mytool` is a few lines of `sh`. It names the tool file in
`SWOOP_TOOL` and runs `swoop`. That variable is all that makes the kit's
launcher your tool: the rows come from the `extensions` folder beside the
file, and every folder takes the file's name. To have `mytool` as a
command, link it into a folder on your `PATH`:

```sh
ln -s "$PWD/mytool/bin/mytool" ~/.local/bin/mytool
```

Nothing of swoop's comes along. The tool has its own config folder, its
own usage log, its own clipboard history, and only the rows its own
extensions print.

## Add an extension

An extension is a folder with one program in it, named the same.
`mytool/extensions/hello/hello` is the whole of one:

```sh
#!/bin/sh
# hello: three rows, a preview for each, and a run that says which one.
# A row is five fields with tabs between: id, kind, icon, title, subtitle.
# SWOOP_TOOL_NAME is the tool this runs under; the launcher sets it.
case "${1:-}" in
  list)
    for n in one two three; do
      printf '%s\tgreeting\t\tHello %s\ta row of %s\n' "$n" "$n" "${SWOOP_TOOL_NAME:-this tool}"
    done ;;
  preview) echo "Enter says hello $2." ;;
  run) echo "hello $2" ;;
  *) echo "usage: hello list | preview <id> | run <id>" >&2; exit 2 ;;
esac
```

`list` prints one row a line, five fields with tabs between: id, kind,
icon, title, subtitle. `preview <id>` prints the right-hand pane.
`run <id>` does the thing. Try it on its own first, where a mistake is
easy to see:

```sh
mytool/extensions/hello/hello list
```

To add one, make a folder beside `hello` with a program of the same name
in it, and make it executable. The launcher finds it the next time it
opens. The extensions in `mytool/extensions` ship with the tool; the ones
a person adds for themselves go in `~/.config/mytool/extensions`, and
join them.

[Writing an Extension](/docs/writing-an-extension) builds one a step at a
time: panes, a ctrl-k menu, a keyword, a key of its own. The rules
themselves, every verb and every row kind, are the contract,
[CONTRACT.md](https://github.com/beatzball/swoop/blob/main/CONTRACT.md)
in the repository.

The kit's programs and its `SWOOP_*` variables keep their names under
every tool, so an extension written for swoop runs in yours unchanged.
One of those variables is the tool's name: the launcher reads it from the
`tool` file and gives it to every extension as `SWOOP_TOOL_NAME`. An
extension that keeps a file of its own puts it in the tool's folder with
it, and never spells a tool:

```sh
recent="${XDG_DATA_HOME:-$HOME/.local/share}/${SWOOP_TOOL_NAME:-swoop}/define/recent"
```

Any of swoop's bundled extensions can be yours the same way: copy its
folder from the repository's `extensions/` into your own.

## Name the frame

The frame is the panel that opens on a hotkey, on a Mac. There is one
frame program, `swoop-shell-mac`, for every tool. It asks the launcher it
runs what tool it is, so the `title` in its menu, the `id` of its launchd
labels and the `hotkey` that opens it are your `tool` file's, with
nothing to build.

Your tool's command installs it:

```sh
mytool/bin/mytool start
```

`start` gives the tool a frame of its own and runs it, now and at every
login:

- a copy of the kit's frame in `~/.local/share/mytool/frame`, signed
  under your `id`, so macOS sees a program of its own: its own entry
  under Accessibility in System Settings, and its own grant
- two launchd agents, `<id>.shell` for the frame and `<id>.clipd` for
  the clipboard watcher, with their logs in `~/Library/Logs/mytool`

```sh
mytool/bin/mytool status     # is it running, where from, on which key
mytool/bin/mytool restart    # install it again: after a new version of the kit
mytool/bin/mytool stop       # until the next login, or the next start
```

It runs beside swoop's frame, and beside any other tool's. Each tool has
its own frame process, its own key, its own folders and its own two
agents, and a `start`, a `stop` or a `restart` of one leaves the others
running: `swoop status` and `mytool status` each show their own. To take
a frame out for good, `stop` it and delete its two files in
`~/Library/LaunchAgents`.

Two tools on one Mac need two keys. `new` writes `ctrl+alt+space` in your
`tool` file, which is not swoop's key. A person changes it for
themselves with `hotkey = ...` in `~/.config/mytool/config`.

The kit's frame is `swoop-shell-mac` in the kit's `bin` folder, beside
`swoop`, in a release and under Homebrew; in a checkout `make shell-mac`
builds it at `shell/mac/.build/release/swoop-shell-mac`, and `start`
finds it there. To try a frame with nothing installed, run it by hand,
with `SWOOP_LAUNCHER` naming the command it runs:

```sh
SWOOP_LAUNCHER="$PWD/mytool/bin/mytool" swoop-shell-mac
```

## Ship it

A release of swoop is one folder in a tarball, and a tool of yours ships
as the same folder with your two parts in it:

```
mytool-0.1.0/
  bin/          the kit's programs and its launcher; on a Mac, the frame
  extensions/   yours
  tool          yours
  VERSION
  LICENSE       the kit's, which is MIT
```

In this layout there is no `bin/mytool`. The launcher looks for `tool`
beside its own `bin` folder, so here `bin/swoop` is your tool, and the
command is a link to it called `mytool`. From a checkout of the kit,
after `make build`:

```sh
kit=/absolute/path/to/swoop
dir=dist/mytool-0.1.0
mkdir -p "$dir/bin"
cp "$kit"/bin/swoop* "$dir/bin/"
cp -R mytool/tool mytool/extensions "$dir/"
cp "$kit/LICENSE" "$dir/"
echo 0.1.0 > "$dir/VERSION"
tar -C dist -czf dist/mytool-0.1.0.tar.gz mytool-0.1.0
```

Check it says who it is:

```sh
dist/mytool-0.1.0/bin/swoop tool name     # mytool
```

The kit's programs are built for one OS and one chip, so there is a
tarball for each. In the repository, `scripts/package` makes swoop's:
it builds the programs for a target, adds the frame on a Mac with the
resource bundles it needs beside it, and adds `scripts/launchd.sh`, which
writes the two launchd agents that run the frame and the clipboard
watcher at login, labelled with your `id`. Once a tool is installed that
way, `mytool start`, `stop`, `restart` and `status` look after its frame.
A tool shipped whole has the frame in its own `bin`, so nothing is
copied: the copy is for a tool that runs the kit's programs from
somewhere else, as the one `new` makes does.

The installer `scripts/get`, the release workflow and the Homebrew
formula that `scripts/tap-formula` writes are swoop's own: they spell
swoop's repository. A tool makes its own, and those are the place to
start. They already read the name, the title and the hotkey from the
`tool` file; what changes is the repository's address, and the tap the
formula goes to.
