---
title: Why I built a launcher the Unix way
date: 2026-10-13
description: A launcher finds, draws and runs. swoop does none of the three itself. fzf finds, libghostty draws, and every extension is a program that prints lines. Here is why.
---

## One row

I use a launcher all day. One key, a few letters, Enter. The leading
launcher does that well, and for years I had nothing to add to it.

Then I did. A list of my runbooks, where Enter runs one. A row that
moves the window I was in to the left half. A pane that asks a model a
question and keeps the answer where I can find it again. Small things,
each one a program I could write in a few minutes.

Each time, the answer was the same. Open the SDK. Learn the component
model. Set up a build. Make a package. The row I wanted was ten lines of
shell. The thing I had to write was an app.

That gap is the reason swoop exists.

## Three jobs

Take the extensions away and ask what a launcher is. It does three
things.

It finds. You type, and a list gets shorter. The best matches rise.

It draws. A panel comes up on a hotkey, over whatever you were doing,
and goes away when you are done.

It runs. You press Enter, and the thing the row stands for happens.

Each of the three had already been done well, by a program I could run.
So swoop does none of them itself.

## fzf finds

fzf reads lines on stdin, lets you filter them, and prints the one you
picked. That is the whole of finding, and it has been tuned for years
by people who use it all day. swoop hands fzf lines and asks for one
back. Every filter, every key, the preview pane and the way the cursor
moves are fzf's. When fzf gets faster, swoop gets faster.

fzf also listens on a socket. That is how an answer that takes time
asks the preview to redraw while it streams in. There is no event loop
in swoop. There is fzf's.

## libghostty draws

On a Mac the panel is a small Swift program. It shows a floating window
on a hotkey, and inside the window is a terminal, drawn by libghostty.
That is all the frame does: show, hide, own the hotkey, run swoop.

The panel is a real terminal, and that buys more than it costs. Notes
open in your editor, inside the panel, and when you quit the list is
back. Icons and previews are pictures, because a terminal that draws
pictures is a thing now. And on Linux or Windows, where there is no
frame yet, the same program runs in any terminal you already have.

What it costs is the look. The panel looks like a terminal, because it
is one. If you want native widgets, this is the wrong tool, and I would
rather say so here than on the issue tracker.

## Every extension is a program that prints lines

This is the part I built for. An extension is a folder with one program
in it, named the same. The program answers four verbs, each one run and
exit:

```
hello list              # one line per result
hello preview <id>      # the right-hand pane for one result
hello run <id>          # do it
hello view <id> [query] # the rows of a pane, for a row of kind "view"
```

A result is one line, five fields, tabs between: an id, a kind, an
icon, a title, a subtitle. The launcher shows the last three and hands
the id back when you pick the row. That is the contract. The whole of it
is one page, with a version number at the top, and `swoop-check` runs
your extension the way the launcher does and names each rule it
breaks.

Ten lines of shell is a working extension. So is a Go binary, a Python
script, or an existing CLI with `sed` on the end of one of its
subcommands. Any language works, as long as it starts fast, because
`list` runs when the panel opens and `preview` runs on every cursor
move.

The bundled extensions are the same kind of program. Apps is one. Ask AI
and Settings were once code inside the launcher; they are extensions
now, and the launcher lost that code. Nothing it does for its own rows
is closed to yours.

Two things fall out of this that I did not plan for.

The first is that you can test an extension without the launcher. Run
`hello list` in a shell and read the lines. Pipe them through `sort`.
The launcher is one more thing that reads stdout.

The second is that every file is a text file. Settings are one line
each in a config file. Snippets and notes are markdown files. Tasks are
one checklist. Quicklinks are tab-separated. You can grep them, sync
them, and edit them in whatever you already use.

## Ask AI is a command too

The same idea reaches the model. Press Tab and the pane opens. Enter
sends what you typed, and the answer streams in on the right. swoop
does not know what a model is. It runs one command with the conversation
on stdin and shows what comes out of stdout, as it comes.

Name the command in a file, or pick one in Settings: a local model, a
hosted one, a coding agent already on your path, or a shell script
that prints a fixed answer.

## Speed is a feature

A launcher that is slow to open is a launcher you stop pressing the key
for. `swoop-nav` and `swoop-preview` run on every keystroke and every
cursor move, so their startup time is felt directly.

So the hot paths are measured. Opening the panel, a keystroke, a cursor
move to the preview drawn, Enter, Esc, and each bundled extension's
`list` are timed on every pull request, each as the smallest of many
runs. Each has a budget, with a reason written beside it, and a pull
request that goes over fails. I would rather argue about a number than
about a feeling.

## A kit, not just a tool

Once every row came from a program, the launcher turned out to be two
things. The programs, the contract and the frame are swoopkit. swoop is
one file, called `tool`, that says a name, a title, an id and a hotkey,
plus a folder of extensions.

So a tool of your own is another file and another folder:

```sh
swoop new mytool
mytool/bin/mytool
```

Three rows, none of swoop's, in folders of its own. On a Mac,
`mytool start` gives it a panel beside swoop's, on its own hotkey. A
picker for a team's runbooks. A launcher for one product. A front end
for a CLI you already have. Anything where a list, a filter and Enter is
the right interface.

## What is next

The Linux and Windows frames. Signed releases, so macOS stops asking for
the Accessibility grant again after each upgrade.

The plan, every design decision and the work in progress are the issues
on GitHub, and the issue is the document. Start there, or start with
the contract and write ten lines.

```sh
brew install beatzball/tap/swoop
```
