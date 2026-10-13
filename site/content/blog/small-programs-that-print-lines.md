---
title: Small programs that print lines
date: 2026-10-13
description: This started as an excuse to build something on libghostty. It turned into a launcher where fzf finds, libghostty draws, and every extension is a program that prints lines.
---

## Where this started

I'll be honest about the order things happened in. I didn't set out to
build a launcher. I'd been using [Ghostty](https://ghostty.org) as my
terminal for a while, and when libghostty showed up, the embeddable core
of it that any app can put a real terminal inside of, I wanted an excuse
to use it for something. A launcher was the excuse. The question I
actually wanted answered was: can you build the whole thing, the panel, the
list, the preview, the lot, out of a terminal and a handful of small
programs, and have it feel as fast and as nice as the launchers people
pay for?

So this project is an experiment first, and a launcher second. The
experiment went better than I expected, and this post is partly a thank
you to the people who made that possible, and partly a look at what
"small programs" turned out to mean in practice.

## libghostty draws

On a Mac the panel is a small Swift program. It shows a floating window on
a hotkey, and inside that window is a terminal drawn by libghostty, the
same renderer Ghostty itself uses. That's the part I didn't have to build,
and it's the part that makes the rest possible. Text is crisp, emoji and
Nerd Font glyphs just work, images in the preview come through as terminal
graphics, and the window shows and hides in a couple of milliseconds. None
of that is mine. I got it by linking a library, through the
[libghostty-spm](https://github.com/Lakr233/libghostty-spm) package that
wraps it up as a Swift dependency with Metal rendering included.

Ghostty's own quick terminal was the template for the panel: a window that
drops down on a key and goes away on the same key. I took the idea, kept it
small, and pointed it at a launcher instead of a shell.

The panel knows nothing about launching. It shows a window, owns the
hotkey, hosts the terminal, and runs swoop in it. The panel and the
launcher talk in two signals, one to show or hide, one from the launcher
to say it's about to exit so the panel can hide first. One bit each way.
I'd like to keep it that way, so a Linux or Windows panel is the same small
thing on a different desktop, with libghostty doing the same job there.

## The three rules

Everything else in swoop follows from an older idea that I didn't want to
lean on without explaining it. The idea is the Unix philosophy, and it
comes down to three rules. A program does one job. Programs talk to each
other in plain text, usually one line at a time, because a line is
something you can read and another program can parse. And because of the
first two, programs can be combined in ways their authors never planned
for.

I think most launchers started out agreeing with this and then grew away
from it. They became one large app with a plugin system, and writing a
plugin meant learning that app. With the drawing handled by libghostty, I
could try the other direction: a launcher made of pieces that each do one
thing. Here's how it turned out, piece by piece, including the places
where I had to bend a rule.

## Finding is fzf's job

fzf reads lines on stdin, lets you type until the list is short, and prints
the line you picked. That is finding, and it's the entire job. swoop gives
it lines and asks for one back. Every keystroke, the filtering, the preview
pane on the right: all of that is fzf doing what it already does well.

Here's the first bend. fzf can also listen on a socket, and I use that for
the AI pane. When you ask a model a question the answer streams in over a
few seconds, and something has to tell fzf to redraw the preview while that
happens. So a small worker posts to the socket. That's a function call, not
a line of text. I'd rather see the answer as it arrives than keep the rule,
so the socket stays.

## Remembering is swoop-nav's job

fzf shows one list. A launcher needs more than one. Your notes are a list,
the actions for a row are a list, a conversation with a model is a list.
Something has to remember which one you're looking at and what was under
the cursor before you pressed Enter. That's all swoop-nav does. fzf runs it
on Enter, on Esc, on typing, on a cursor move, and it prints back the fzf
actions to take next. Those are lines too, which pleases me more than it
probably should.

The bend here is obvious once you see it. A program that starts, prints,
and exits can't remember anything on its own. So swoop-nav keeps the stack
of open panes in a small file between keystrokes. That's state on disk. I
can argue that it's a file you never see and never have to care about, and
that's true, but it's still a file.

## Showing and running are two small programs

When the cursor lands on a row, fzf runs swoop-preview with the row's id.
It figures out which extension the id belongs to, asks that extension for a
preview, and prints whatever comes back. That's the whole program.

When you press Enter, fzf hands the id to swoop-run, which asks the
extension to do the thing. swoop-run prints nothing, because its result
isn't text. It's a window opening, or a task ticked off, or an emoji landing
in whatever you were typing. fzf actually replaces itself with swoop-run at
that point, so once your app is up there's nothing left running behind it.

Two more bends live here. The preview pane's size arrives in environment
variables, not as an argument, because that's how fzf passes it. And a
preview isn't always text. An app icon or an image preview goes out as
terminal graphics escapes. That's still a line, strictly speaking, but you
won't enjoy grepping it.

## Matching inside a view

Once you're inside a view, fzf's own matching is turned off. The extension
decides which rows exist as you type, and one of them might be "add a task
with this text", which fzf has never seen. So the extension prints all its
rows and pipes them through swoop-match with whatever you've typed. Rows
in, matching rows out, best first. That is a filter in the oldest sense of
the word.

The honest objection is that finding was supposed to be fzf's job, and now
two programs do it. I think it's the right call, because a view that makes
rows from your text can't hand the matching back to fzf. But I'm counting
it as a bend.

## Extensions, which is the part I actually care about

An extension is a folder with one program in it, named the same as the
folder. The program answers a handful of verbs. `list` prints one line per
result. `preview` prints the pane for one result. `run` does it. A result is
one line with five tab-separated fields.

That's it. There's no package to build, no manifest, no SDK to install. The
first extension in the repository is still a 50-line shell script, and
sixteen others followed it in Go and bash without the contract needing to
change much. You can also test an extension without the launcher running:

```sh
hello list | sort
```

That works because the launcher is just one more program reading stdout,
and your extension doesn't know or care who's on the other end.

The bend here is mine to own. Several verbs in one program is, strictly,
several jobs in one program. I chose that on purpose. An extension is the
thing a person writes in ten minutes, and one file is what you write in
ten minutes.

## So does it hold up?

Mostly, I think. The rules hold where they matter most, at the boundary
between the launcher and your program. They bend where something had to
remember, draw a picture, or stop. I've written each bend down so the next
one has to be argued for instead of slipping in while nobody's looking.
Every setting is a line in a text file, and you can grep all of it.

And the experiment? A terminal plus a handful of small programs really is
enough for a launcher that feels native, which I half expected and half
didn't. Most of the credit for the "feels native" part goes to libghostty.
If you're a Ghostty user, you already know why. If you're not, this is a
good reason to go look.

If you'd rather check my work than take my word for it, the contract fits
on one page, and ten lines of shell will tell you whether I'm right.
