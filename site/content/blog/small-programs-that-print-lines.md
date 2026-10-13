---
title: Small programs that print lines
date: 2026-10-13
description: fzf finds. libghostty draws. Every extension is a program that prints lines. The three old rules behind that, and whether each of swoop's programs actually keeps them.
---

## The slogan problem

The README says swoop is small programs that do one job and talk in
lines of text. I wrote that line, I like it, and I've started to notice
it's doing a lot of work for something nobody has asked me to prove. So
let's prove it, or find out where it's a stretch.

The rules I mean are the old ones, the Unix philosophy as it's usually
told. A program does one job. Its interface is text, one line at a time,
because a line is something a person can read and another program can
parse. And programs work together, which is less a rule than what
happens when you keep the first two.

A launcher, once you take the extensions away, finds, draws and runs.
swoop does none of the three itself, which is the whole trick. fzf
finds. libghostty draws. Your programs run. Here's how each piece holds
up, and where it doesn't.

## fzf

fzf reads lines on stdin, lets you type until the list is short, and
prints the one you picked. That's finding, the entire job. swoop hands
it lines and asks for one back; every filter, every key, the preview
pane, all fzf's.

The honest part: fzf also listens on a socket, and I use it. When you
ask a model a question the answer takes a few seconds, and something has
to tell the preview to redraw while it streams in, so a worker posts to
that socket. That's a call, not a line. I'd rather see the answer before
it's finished than keep the rule.

## swoop-nav

fzf shows one list. A launcher wants more than one — your notes, the
actions for a row, a question with a running answer — and something has
to remember which pane you're in. That's swoop-nav's whole job. fzf runs
it on Enter, Esc, typing, a cursor move, and it prints back the fzf
actions to perform, which are, pleasingly, lines.

Where it bends: a program that runs and exits can't remember anything,
so swoop-nav keeps the stack of panes in a file between keystrokes.
That's state on disk, and I'm not going to pretend a file you never see
is the same as no file.

## swoop-preview

Every time the cursor lands on a row, fzf runs swoop-preview with the
row's id. It works out which extension the id belongs to, strips the
prefix, hands over, and prints whatever comes back. One hop, no
opinions.

The size of the pane arrives in two environment variables fzf sets, not
as an argument. And what comes back isn't always text: an icon or a
preview can be a picture, sent as terminal graphics escapes. A line
that's a picture is still a line, technically, but try grepping it.

## swoop-run

Enter hands the picked id to swoop-run, which finds the extension and
runs its `run` verb, and then the thing happens. It doesn't print
anything back, and that's the bend. fzf starts it with `become`, which
replaces fzf with swoop-run outright, so when your app is up there's
nothing left running behind it. It's the end of the pipe, on purpose,
and the one program here whose output isn't lines, because its output is
the world changing a bit.

## swoop-match

Inside a view, fzf's matching is switched off, because the extension
decides what rows exist as you type, and one of them may be a row for
what you just typed, which fzf has never seen. A shell extension prints
every row and pipes them through swoop-match with the text from the bar.
Rows in, matching rows out, best first. A filter in the old sense.

Its sin is existing. Finding was fzf's job, and now two programs do it.
I think it's the right call, but I'm counting it.

## The extensions

This is the part I actually built the thing for. An extension is a
folder with one program in it, named the same, and the program answers a
few verbs: `list` prints a line per result, `preview` prints the pane
for one, `run` does it. A result is one line, five fields, tabs between.

I remember the first one. The plan was a proper Go extension to check
the contract against. Instead I typed a shell script, three printf lines
and a case statement, and ran the launcher — and there were my three
rows, with a preview, and Enter did the thing. Nothing to build, no
package, no manifest. I sat there for a bit. It's the moment I stopped
worrying about whether the idea worked and started worrying about
everything else.

```sh
hello list | sort
```

That's the other thing you get: you can test an extension without the
launcher, because the launcher is just one more program reading stdout.

Where it bends, and this one's on me: several verbs in one program is
several jobs in one program. I picked one, because an extension is the
thing a person writes, and one file is what a person writes in ten
minutes.

## The frame

On a Mac the panel is a small Swift program. It shows a floating window
on a hotkey, puts a terminal drawn by libghostty inside it, and runs
swoop there. It knows nothing about launching.

The frame and the launcher don't talk in lines at all. They talk in
signals: one to toggle the panel, one from the launcher to say it's
about to exit so the panel can hide first. It's one bit each way, which
is kind of all the two have to say to each other.

## So, does it earn it?

Mostly, I think. The rules hold where they matter, at the line between
the launcher and your program. They bend where a program had to
remember, draw a picture, or stop, and I've written each bend down so
the next one has to be argued for rather than slipped in. Every setting
is a line in a file, and you can grep the lot.

If you want to check my working, the contract is one page, and ten lines
of shell is enough to find out.