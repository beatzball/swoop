---
title: Extensions
description: Everything that ships in the list, from quicklinks and emoji to notes, tasks, window moves and clipboard history. Each one is an extension.
sidebar:
  order: 3
---

## Every row is an extension

Apps come first. Everything else in the list comes from an extension: a
small program in the repository's `extensions/` folder that prints rows.
Yours sit beside them in `~/.config/swoop/extensions/`, and work the same
way. [Writing an Extension](/docs/writing-an-extension) builds one from
nothing.

Any extension can be turned off in [Settings](/docs/settings), under
Extensions. ctrl-k on a row shows what else that row can do.

| type | what you get | extension |
|---|---|---|
| `2+2` | the answer, Enter copies it | calc |
| `wiki`, `google`, your own names | a quicklink | links |
| `emoji` | every emoji, flag and symbol | emoji |
| a snippet's name or keyword | the snippet, pasted | snippets |
| `notes` | your markdown notes | notes |
| `tasks` | your checklist | tasks |
| `reminders` | Apple Reminders, on a Mac | reminders |
| `clipboard` | what you copied | clipboard |
| `search files` | files by name | files |
| `define` | a dictionary | define |
| `left half`, `maximize` | window moves | window |
| `stats` | what you open most | stats |
| `sleep`, `lock` | a few things a Mac can do | system |
| `settings` | [Settings](/docs/settings) | settings |
| Tab | [Ask AI](/docs/ask-ai) | ai |

## Quicklinks

A quicklink is a name, a link, and what opens it. Type the name, press
Enter, and the link opens: a URL, a folder, a file, a deeplink another app
owns. A link that holds `{argument}` asks first: Enter opens a pane, what you
type goes into the link, encoded, and Enter again opens it. Five ship ready
to use: Google, DuckDuckGo, Wikipedia, YouTube, GitHub. Type `wiki`, Enter, a
term, Enter.

They are one file, `~/.config/swoop/quicklinks.tsv`: name, link, app,
tab-separated, one per line. Edit it, or let the tool:

```sh
swoop-links add 'Home' ~ Finder
swoop-links add 'npm' 'https://www.npmjs.com/search?q={argument}'
swoop-links import ~/Downloads/quicklinks.json   # a JSON export: name, link, openWith
swoop-links defaults > ~/.config/swoop/quicklinks.tsv   # start from the five
```

ctrl-k on a quicklink copies the link, filled in, or deletes it.

## Emoji and symbols

Type `emoji`, Enter, then a name or a keyword: `rocket`, `+1`, `flag jap`,
`arrow`, `euro`, `alpha`. Every emoji and flag is there, and a few hundred
symbols: arrows, math, currency, punctuation, keyboard keys, box drawing,
Greek. What you used last comes first.

Enter pastes into the app you came from. The frame hides, then presses
cmd+V, but only when a text field has the focus (or the app shows none, like
a terminal); on a button or the desktop it copies and a notification says
so. The keystroke needs Accessibility for swoop-shell-mac: the first paste
brings up the system dialog, and until it is on, Enter copies. In a plain
terminal, and on Linux for now, Enter copies.

ctrl-k copies, or pastes in one of the six skin tones; the default tone is in
Settings. Words of your own go in `~/.config/swoop/emoji.keywords`, one line
per character: `😀 grin happy`.

## Snippets

A snippet is named text you paste often: a signature, an address, a reply.
Type its name, press Enter, and it is pasted into the app you came from, the
same way as emoji. Each snippet is a markdown file in
`~/.config/swoop/snippets/`: the first line is the name, an optional
`keyword:` line follows, and the rest is the text.

```text
Signature
keyword: ;sig

Best,
{cursor}
```

`{date}`, `{time}`, `{clipboard}` and `{uuid}` are filled when you paste, and
`{cursor}` is where the caret ends up. To bring snippets from another
launcher, or write one from the shell:

```sh
swoop-snippets import ~/Downloads/snippets.json   # a JSON export: name, text, keyword
echo 'Thanks, {clipboard}' | swoop-snippets add 'Thanks' ';ty'
```

## Notes

Type `notes`, Enter: your notes, the most recently changed first, each
previewed as rendered markdown. Typing searches the titles and the text.
Enter opens the note in your editor, right in the panel; quit the editor and
the list is back, the preview showing your change.

`New note`, at the top, makes a note whose first line is what you typed, and
opens it the same way; with nothing typed, the note is named by the date and
time, `2026-09-28 21:15`. The editor is the Editor setting (`editor = nvim`
in the config), else `$EDITOR`, else `nano`. ctrl-k opens the note in the app
that opens `.md` files instead, copies the text, shows the file in its
folder, or deletes it. An editor wants room: widen the list with the divider
keys, or drag the panel's edge.

Each note is a markdown file in `~/.local/share/swoop/notes/`, and its first
line is its title. Any editor works, and so does any sync. Delete moves the
file to `deleted/` in that folder, so you can get it back.

## Tasks

Type `tasks`, Enter. The list is open tasks first, the ones due soonest on
top, done ones after. Type a task and press Enter to add it. End it with
`today`, `tomorrow`, a weekday, or a date like `2026-10-01`, and that is its
due date: `buy milk tomorrow`. Enter on a task ticks it, and you stay in the
list. ctrl-k undoes, deletes, or copies, and `Edit the list` opens the whole
file in your editor, in the panel.

The tasks are one markdown checklist, `~/.local/share/swoop/tasks.md`, so any
editor can change it too:

```text
- [ ] buy milk due: 2026-09-29
- [x] call the bank
```

## Reminders

On a Mac, `reminders` is the same view over Apple Reminders: every list,
overdue and today first. Enter completes a reminder; typed text adds one to
your default list, with the same date words as Tasks. The first time, macOS
asks to let swoop use Reminders.

## Clipboard History

Type `clipboard`, Enter: what you copied, newest first. Enter puts it back
on the clipboard; ctrl-k deletes it from the history.

The history is kept by `swoop-clipd`, the one thing that runs while the
launcher is closed. The launcher starts it the first time and it keeps
running, polling the clipboard a few times a second and appending new text to
`~/.local/share/swoop/clipboard/history.jsonl`, readable by you only.

Two things it never keeps: a copy that carries the "concealed" mark some
password managers set, and a copy made while an app on the ignore list is in
front. The list defaults to the known password managers; write your own, one
bundle id per line, at `~/.config/swoop/clipboard.ignore`. Copy a password
from your manager and run `swoop-clipd types` to see what it marks, and
`osascript -e 'id of app "Its Name"'` to get its bundle id.

If a manager still gets through, start the watcher with
`SWOOP_CLIPD_DEBUG=1` and read `watcher.log` next to the history: each change
is logged with the app in front and the marks seen, never the text.

`swoop-clipd status` says whether it is running; `swoop-clipd delete <id>`
and `swoop-clipd clear` remove entries. Stop it if you would rather it did
not run.

## Search Files

Type `search files`, Enter, then part of a name. On a Mac the names come from
Spotlight, under your home folder, newest change first; with nothing typed,
the files you used most recently. The preview shows a picture as a picture,
text as text, a folder as its listing. Enter opens the file in its default
app; ctrl-k Edit, on a text file, opens it in your editor inside the panel.

## Calculator

Type arithmetic, `2+2` or `(17*3)/4`, and the answer is a row: `12.75`. Enter copies
it. The arithmetic is done by `bc`, which ships with macOS and every Linux.

## Define Word

Type `define`, Enter, then a word. With nothing typed, the words you looked
up last. The preview is the definition, from the system dictionary; Enter
opens the word in Dictionary.

## Window management

Rows that move the window you were in: Left, Right, Top, and Bottom Half;
First, Center, and Last Third; First and Last Two Thirds; the four quarters;
Maximize, Almost Maximize, Reasonable Size, Center; Next and Previous
Display. Type `left half`, Enter. The launcher's panel does not take focus,
so the app you came from is the one that moves. The preview names the window
and the frame it will get.

On macOS this needs Accessibility. The first time, Enter opens System
Settings, Privacy & Security, Accessibility: turn on swoop (or the terminal
swoop runs in), and run the row again. Other systems come with their frames.

## Used recently, and Stats

The list starts with the five things you opened most recently, marked
"recent", then everything else in its usual order. Every Enter that opens
something is one line in `~/.local/state/swoop/usage.jsonl`, yours alone and
never sent anywhere. Type `stats` for the whole picture: what you open most,
how often, when last, and a fortnight by day. ctrl-k there clears the log;
Settings turns the group off.

## System

Sleep, Lock Screen, Toggle Dark Mode, Empty Trash, Start Screensaver. It is
one short shell script, `extensions/system/system`, and the smallest complete
extension in the repository: read it next to
[Writing an Extension](/docs/writing-an-extension).
