---
title: The Extension Contract
description: The contract between swoopkit's launcher and an extension, with a version number, a compatibility promise, and swoop-check, the test an extension author runs.
sidebar:
  order: 6
---

**contract 1**, 2026-10-02

This is the contract between swoopkit's launcher and an extension: what
the launcher promises, and what an extension must do. It is the current
contract, whole, in one place. How each part came to be, with its date
and its reason, is the history in
[issue #2](https://github.com/beatzball/swoop/issues/2).

An extension is a program that prints lines. The launcher runs it, shows
the lines as rows, and hands the picked row's id back to it. Any language
works; the examples here are shell.

`swoop-check`, at the end, is the test: it runs an extension the way the
launcher does and says which rule it breaks.

## The version

The number at the top is the contract's version: a major number, and a
minor one after the first addition (`1.1`).

- **Within a major version, an extension that met the contract keeps
  working.** A row it prints is shown the same way, a verb is asked with
  the same arguments, a file beside it is read the same way
- **An addition is a new minor version**: a new verb, kind, file,
  variable or setting. It gets a dated line under [Changes](#changes).
  An extension that does not use it is not touched by it. A launcher
  skips what it does not know (a setting, a key in the `tool` file, a
  line that is not a row), so a file written for a newer minor still
  reads under an older one
- **A removal, or a changed meaning, is a new major version**
- Before swoop 1.0 the major may move, and swoop's changelog says so when
  it does. From swoop 1.0 on, the contract's major moves only with
  swoop's own

Everything in this document is in the contract. What is not in it is not
promised, and [What is not in the contract](#what-is-not-in-the-contract)
names what looks as if it were.

## An extension in ten lines

```sh
#!/bin/sh
# hello: three greetings. Enter says the one under the cursor.
set -eu
row() { printf '%s\t%s\t%s\t%s\t%s\n' "$1" text '' "$2" "$3"; }
case "${1:-}" in
  list)    row en 'Hello' 'English'; row fr 'Bonjour' 'French'; row es 'Hola' 'Spanish' ;;
  preview) echo "Enter says the greeting: ${2:-}" ;;
  run)     echo "greeting: ${2:-}" ;;
  *)       echo "usage: hello list | preview <id> | run <id>" >&2; exit 2 ;;
esac
```

Save it as `hello` in a folder called `hello`, where the launcher looks,
and make it a program:

```sh
mkdir -p ~/.config/swoop/extensions/hello
cp hello ~/.config/swoop/extensions/hello/hello
chmod +x ~/.config/swoop/extensions/hello/hello
swoop-check ~/.config/swoop/extensions/hello
```

`swoop-check` prints `hello: meets contract 1`. Run `swoop` in a
terminal, type `bonj`, press Enter: the launcher closes and the terminal
shows `greeting: fr`.

[Writing an Extension](https://swoop.sh/docs/writing-an-extension) builds
a larger one a step at a time.

## Where an extension lives

An extension is a folder that holds one program with the folder's name:

```
~/.config/swoop/extensions/hello/hello
```

**The launcher** looks in these folders, in this order, and the first
folder to have a name wins:

1. `~/.config/<name>/extensions`, where `<name>` is the tool's name from
   [the `tool` file](#the-tool-file), `swoop` for swoop.
   `$XDG_CONFIG_HOME` stands in for `~/.config` when it is set
2. the `extensions` folder beside the `tool` file: the ones the tool
   ships
3. each folder in `$SWOOP_EXTENSIONS`, separated by `:`

So an extension of the user's replaces a bundled one of the same name. A
folder with no program of its own name is not an extension and is passed
over without a word: a README beside the extensions is fine. An extension
the user turned off in Settings is asked nothing: no rows, no view, no
key.

**The extension** must be a file the system can run: the execute bit set,
and a `#!` line if it is a script. The name is the folder's name.

Every time the launcher runs the program:

- it is one process that answers one verb and exits
- the working folder is the extension's own folder, and
  `$SWOOP_EXT_DIR` holds its path, so files that ship beside the program
  are found
- the kit's programs (`swoop-match` and the rest) are on `PATH`

## The line

A row is one line on stdout: five fields with a tab between them.

```
id	kind	icon	title	subtitle
```

| field | what it is | shown |
|---|---|---|
| `id` | the row's name, in the extension's own terms. The launcher hands it back, as it is, to `preview`, `run`, `view`, `actions` and `send` | no |
| `kind` | one word. It says what Enter does: see [The row kinds](#the-row-kinds). A word the launcher does not know is the extension's own, and Enter runs the row | no |
| `icon` | one glyph, an emoji or a Nerd Font character. May be empty | yes |
| `title` | the main text. At the root, what is typed is matched against it | yes |
| `subtitle` | dimmer text after the title. May be empty | yes |

```sh
printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$kind" "$icon" "$title" "$subtitle"
```

**The extension** must:

- print every row as exactly five fields: four tabs, also when a field
  is empty
- keep tabs and newlines out of every field. A tab inside a title makes
  a sixth field, and the line is no longer a row. Put a space in its
  place
- print UTF-8
- give every row an id that is not empty, that is the same the next time
  the same thing is listed, and that no other row in the same list has
- keep a line under 1 MiB

**The launcher** promises:

- an id comes back byte for byte. It may hold spaces, slashes, anything
  but a tab or a newline
- ANSI colour escapes in the icon, the title and the subtitle are drawn
- a line that is not a row is skipped, and the rows around it still
  show. A line over 1 MiB ends the reading: that row and the ones after
  it are lost
- at the root, the rows of every extension are one list, in order of
  title, and what is typed is matched against the title, never the
  subtitle. In a view, the rows show exactly as printed, in that order

## The verbs

```
hello list [text]            print the rows for the root list
hello preview <id>           print the preview for one row
hello run <id> [action]      do the row, or one action on it
hello view <id> [text]       print the rows of a pane
hello actions <id>           print the menu for one row
hello send <id> <text>       take the text typed in a prompt
```

`list`, `preview` and `run` are the whole of a plain extension. `view` is
for an extension with a row of kind `view`. `actions` is optional. `send`
is for a view whose bar is a prompt. The id is always the extension's
own, as it printed it.

### list

**The launcher** asks `list` when it opens, with no text. At the root it
asks again on every keystroke, with the text in the bar as one argument,
and with no argument when the bar is empty. It reads stdout to its end.
It gives `list` two seconds.

**The extension** must print its rows and exit 0 inside two seconds, with
a text and without one. No rows is an answer: print nothing and exit 0.

- A list that does not depend on the text ignores the argument. The
  launcher does the matching
- A row made from the text must have the text in its title, or the
  matching at the root hides it. A calculator's row for `2+2` is titled
  `2+2 = 4`
- An exit other than 0, or a `list` still running at two seconds, shows
  none of that extension's rows, and puts one line on the launcher's
  stderr. The other extensions' rows still show
- A program that `list` leaves running must not keep `list`'s stdout or
  stderr open: the launcher reads until both close. Give it
  `< /dev/null > /dev/null 2>&1 &`

### preview

**The launcher** runs `preview <id>` each time the cursor comes to a row,
any row: a row of kind `view` too. It shows what the program prints, ANSI
colours and Kitty graphics included. `$FZF_PREVIEW_COLUMNS` and
`$FZF_PREVIEW_LINES` hold the size of the preview window. There is no
time limit, and a preview still running when the cursor moves on is
stopped.

**The extension** must print the preview and exit 0 for every id it
printed. It runs on every cursor move, so it must be fast; keep what is
slow in a cache of your own. After an exit other than 0 the launcher adds
a line that says so under whatever was printed.

### run

**The launcher** runs `run <id>` on Enter, and `run <id> <action>` for an
action picked from the row's menu, where `<action>` is the id of the
action's row. What the launcher does after is the row's kind:
[The row kinds](#the-row-kinds).

**The extension** must do the thing and exit 0 when it worked.

For a row that closes the launcher, in a plain terminal the run has the
terminal: what it prints is there once the launcher is gone. Inside the
frame the panel hides and the output is not seen, so say what matters
some other way.

### view

**The launcher**, on Enter on a row of kind `view`, opens a pane: the
row's title is the prompt, the bar is empty, and the rows are what
`view <id>` prints, where `<id>` is that row's own id. On every keystroke
it asks `view <id> <text>` and shows exactly what comes back, in that
order: its own matching is off inside a view. It gives `view` two
seconds, and takes the answer whole, when the program exits.

- Esc with text in the bar clears the text. Esc with an empty bar leaves
  the pane, and the pane below comes back with the text and the cursor
  row it had
- The rows of a view are rows like any other: `preview`, `run` and
  `actions` work on them, and a row of kind `view` among them opens a
  further pane
- After typing, the cursor is on the first row

**The extension** must print the pane's rows and exit 0 inside two
seconds, with a text and without one. It does the filtering and chooses
the order. No match is an answer: print nothing and exit 0.
[`swoop-match`](#swoop-match) filters the way the bundled views do:

```sh
view)
  # $2 is the view's id, $3 the text in the bar.
  all_rows | swoop-match "${3:-}"
  ;;
```

The same holds as for `list`: an exit other than 0 or two seconds gone
shows no rows, and nothing left running may hold the output open.

### actions

**The launcher**, on ctrl-k on a row, asks `actions <id>`. When rows
come back it opens them as a menu, titled with the row's title; typing
filters the menu by title, and the preview stays on the row the menu is
for. Enter on an action runs `run <id> <action>`, with the id of the row
the menu is for and the id of the action's row. It gives `actions` two
seconds.

**The extension** may answer `actions` or not. No rows, or an exit other
than 0, is a row with no menu: ctrl-k does nothing. A row it prints has
the action's name as its id, and one of three kinds:

```sh
actions)
  printf '%s\t%s\t%s\t%s\t%s\n' copy   action  '' 'Copy'   'Back onto the clipboard'
  printf '%s\t%s\t%s\t%s\t%s\n' delete refresh '' 'Delete' 'Remove from history'
  ;;
run)
  case "${3:-copy}" in
    delete) forget "$2" ;;
    *)      copy "$2" ;;
  esac
  ;;
```

### send

`send` is Enter in a view whose bar is a prompt. It is under
[`views`](#views).

## The row kinds

The kind says what Enter does. "Quietly" means the run's output is not
shown and the launcher stays open.

| kind | on a row | in the menu `actions` prints |
|---|---|---|
| `view` | opens a pane filled by `view <id>` | not used |
| `action` | as any other word, the last line | `run <id> <action>`, then the launcher closes |
| `refresh` | in a view: `run <id>` quietly, then back to the pane below, reloaded | `run <id> <action>` quietly, then back to the pane the menu was opened from, reloaded |
| `toggle` | in a view: `run <id>` quietly, and the pane stays: reloaded, the bar cleared, the cursor on the same row number | not used |
| `terminal` | `run <id>` with the whole terminal, keys and screen. When it exits the launcher is still there: the same pane, reloaded, the bar's text and the cursor's row kept, the preview drawn again | `run <id> <action>` with the whole terminal, then back to the pane the menu was opened from, reloaded, the preview drawn again |
| `group` | a header over the rows below it. Enter does nothing | not used |
| any other word | `run <id>`, then the launcher closes | as `action` |

- `refresh` is for a row that changes something and is done: a setting's
  choice, Delete in a menu. `toggle` is for a row that changes itself
  and stays in the list: a task ticked off. At the root both run and
  close the launcher, as any other word does
- `terminal` is for a program that needs the keyboard: an editor on a
  note

**A landing.** The run of a `terminal` row, and a `send`, may say which
row the cursor comes back to. Write that row's id, as the extension
prints it, to the file named by `$SWOOP_LAND` before exiting. The
launcher then clears the bar, reloads the pane, and puts the cursor on
that row, or on the first row when the id is not in the list. Without
the file nothing changes.

```sh
run)
  note="$(new_note "$2")"
  "${EDITOR:-vi}" "$note"
  printf '%s' "$(basename "$note")" > "$SWOOP_LAND"
  ;;
```

**A header.** A row of kind `group` is drawn as a header: its title at
the left edge, in the icon's column, dimmed, with its subtitle kept. The
cursor never rests on it: a pane opens with the cursor on the first row
that is not a header, Up and Down step over one, and a click on one does
nothing. The extension must leave the header's icon empty, and must not
print a header with no row under it.

```sh
printf '%s\t%s\t%s\t%s\t%s\n' g/today group '' 'Today' '2 tasks'
printf '%s\t%s\t%s\t%s\t%s\n' t/1 toggle '' 'Water the plants' ''
printf '%s\t%s\t%s\t%s\t%s\n' t/2 toggle '' 'Send the report' ''
```

## The files beside the program

Five small files in the extension's folder tell the launcher something
before it runs anything. Each is optional. The launcher reads them; the
extension only ships them.

### keyword

One word, on the first line.

```
def
```

**The launcher**: that word and a space, at the start of the root bar,
scope the bar to the extension. `def ap` asks only this extension, with
`ap`.

- When the extension claims a key (see `key`), the view that key opens
  is opened, with the rest of the bar as its text
- Otherwise, when its `list` is one row and that row is of kind `view`,
  that view is opened, with the rest of the bar as its text
- Otherwise the root shows only this extension's rows: `list <rest>`,
  kept where every word of the rest matches the title or the subtitle

The word alone, with no space, is nothing special: it may also be the
start of a row's title. Case does not count. When two extensions have
the same keyword, the first by name has it.

### key

One claim per line: a key, the id of the view it opens, and the view's
title.

```
tab ask Ask AI
```

**The launcher**: the key opens that view from anywhere, as Enter on a
`view` row with that id would; the title is the pane's prompt. Inside
that view the key does nothing. The view needs no row of its own in
`list`.

- The keys that may be claimed: `tab`, `shift-tab`, `f1` to `f12`, and
  `alt-` with one lower-case letter, one digit, `,`, `.` or `/`
- When two extensions claim one key, the first by name has it.
  `swoop status` lists who has each key, and every claim that lost
- An extension turned off gives its keys up
- A view whose bar is a prompt (see `views`) opens with the bar's text
  kept, as the text to send. Any other opens with an empty bar, and the
  text is back in the bar on the way out
- A blank line, and a line that starts with `#`, are skipped. With no
  title, the extension's name is the prompt

**The extension** must answer `view <view-id>`.

### views

One view per line: the view's id, then its settings, with spaces
between.

```
ask bar=prompt preview=wrap,follow
```

A view not named here is a plain list. A blank line, a line that starts
with `#`, and a setting the launcher does not know are skipped.

**`bar=prompt`**: the bar is text to send, not a filter.

- Typing changes nothing in the list. `view <id>` is asked with no text
- Enter, with text in the bar, runs `send <row-id> <text>`: the id of
  the row under the cursor, and the text. Enter with an empty bar does
  nothing, and Enter never runs a row in this view
- Exit 0: the bar is cleared and the rows are listed again, with the
  cursor on the row named in `$SWOOP_LAND`, or where it was when no row
  is named. Any other exit leaves the text in the bar
- `send` has two seconds

**The extension** must answer `send`, and its view must print at least
one row, since the text goes to a row. An answer that takes longer than
`send` may is the job of a worker that `send` starts and leaves. The
worker tells fzf when there is something new to show, through fzf's own
HTTP API: the action as the body of a POST, to the socket in `$FZF_SOCK`
(a local port in `$FZF_PORT` on Windows), with the key in `$FZF_API_KEY`.
`refresh-preview` draws the preview again. `reload(swoop-nav rows)`
lists this pane again; in a view whose bar filters, the text goes along:
`reload(swoop-nav rows {q})`.

```sh
send)
  # $2 is the row under the cursor, $3 the text.
  (
    answer "$2" "$3" > "${TMPDIR:-/tmp}/hello-answer.txt"
    [ -z "${FZF_SOCK:-}" ] || curl -s -m 2 --unix-socket "$FZF_SOCK" \
      -H "x-api-key: ${FZF_API_KEY:-}" --data-binary 'refresh-preview' http://fzf/
  ) < /dev/null > /dev/null 2>&1 &
  ;;
```

**`preview=`**: the view's own preview window, a list with commas: a
width from `20%` to `80%`, `wrap` to wrap long lines, `follow` to keep
the end of a growing text in view. It holds while the view and its
actions menu are open, and the window the user had comes back after.

### cache

The word `run`.

**The launcher**: `list` is asked once per launch, with no text, and the
rows are kept until the launcher closes. Typing at the root only matches
the kept rows. A `list` that failed is asked again at the next reload.
The keyword still asks `list <text>` each time.

**The extension**: this is for a long list that does not depend on the
bar and does not change while the launcher is open, the applications on
the machine.

### icons

The word `id`. It counts only beside a `cache` file that holds `run`.

**The launcher**: each row shows the icon of the file its id names, in
place of its glyph. The glyph is the fallback. Today only a macOS
application bundle gives an icon.

**The extension** must make each row's id the path of a file.

## The variables

The kit's variables carry its own prefix under every tool, so an
extension written for one tool runs under another.

| variable | set for | what it holds |
|---|---|---|
| `SWOOP_EXT_DIR` | every verb | the extension's own folder, which is also the working folder |
| `SWOOP_EXTENSIONS` | every verb | the extra folders of extensions the launcher searches, with `:` between; the tool's bundled ones are first |
| `SWOOP_TOOL` | every verb | the path of [the `tool` file](#the-tool-file) of the tool that is running |
| `SWOOP_LAND` | `run` and `send` | the file to write a row id to, for [a landing](#the-row-kinds) |
| `SWOOP_PASTE` | `run`, inside the frame only | the file to leave a paste request in; not set in a plain terminal |
| `FZF_SOCK` | every verb fzf starts | fzf's socket, for a worker; `FZF_PORT` on Windows |
| `FZF_API_KEY` | every verb | the key a request to that socket carries, in the header `x-api-key` |
| `FZF_PREVIEW_COLUMNS`, `FZF_PREVIEW_LINES` | `preview` | the size of the preview window |

- The first `list` of a launch runs before fzf does, with no `FZF_SOCK`.
  A worker checks that it is set, and does without when it is not: the
  next keystroke shows the new rows
- **A paste.** An extension whose Enter puts text where the user was
  typing copies the text to the clipboard, and, when `SWOOP_PASTE` is
  set, leaves a request there: a first line with a count, then the text,
  byte for byte, with no newline added. The frame, once its panel is
  gone, sends the paste keystroke to the app in front when a text field
  is there, then presses the left arrow that many times; the count is
  `0` for none. Write the file whole, under another name and then
  renamed. Without `SWOOP_PASTE` the text is on the clipboard and the
  user pastes it

```sh
printf '%s' "$text" | pbcopy
if [ -n "${SWOOP_PASTE:-}" ]; then
  printf '0\n%s' "$text" > "$SWOOP_PASTE.tmp" && mv "$SWOOP_PASTE.tmp" "$SWOOP_PASTE"
fi
```

## The limits

| limit | what it bounds | past it |
|---|---|---|
| 2 seconds | `list`, `view`, `actions`, `send`, each call | the program is stopped; no rows, no menu, or the text left in the bar |
| 1 MiB | one line of rows | the reading ends there |
| 4096 bytes | the `tool` file | the file is not read |
| `ext/<name>/` | the prefix of every id | see below |

**The id prefix.** The launcher owns the routing. Inside the launcher
every id is `ext/<name>/<id>`: the extension's name, then the id the
extension printed. The launcher adds the prefix as the rows come in and
takes it off before the id goes back, so the extension never sees it and
must not print it. `swoop-run` and `swoop-preview`, the kit's programs
that hand an id to its extension, take the whole id and refuse one
without the prefix. The file in `$SWOOP_LAND` takes the id with no
prefix.

`preview` and `run` have no limit of time.

## The tool file

A tool built on the kit is a folder with an `extensions` folder and,
beside it, a file called `tool`: a key, a space and a value on each
line.

```
name mytool
title My Tool
id com.example.mytool
hotkey alt+space
```

| key | what it names | without it |
|---|---|---|
| `name` | the word in the folders (`~/.config/<name>`, `~/.local/share/<name>`, `~/.local/state/<name>`, the logs) and the command | `swoop` |
| `title` | what the frame and a notification show | the name |
| `id` | the prefix of the launchd labels, `<id>.shell` and `<id>.clipd`, and of the frame's signature | `dev.<name>` |
| `hotkey` | the key that opens the frame until the user picks one | `alt+shift+space` |

- `name`, `id` and `hotkey` are one plain word: letters, digits, and
  `.` `_` `-` `+` after the first character. A value that is not is
  skipped
- A blank line, a line that starts with `#` and a key the launcher does
  not know are skipped
- A file over 4096 bytes is not read
- The launcher finds the file at `$SWOOP_TOOL`, or at `../tool` from its
  own `bin/`. It exports `SWOOP_TOOL` to everything it runs, and takes
  the bundled extensions from the folder beside the file
- `swoop tool` prints the four lines, filled in. `swoop tool <key>`
  prints one value

A tool of your own is that folder and a two-line program:

```sh
#!/bin/sh
SWOOP_TOOL="$(dirname "$0")/tool" exec swoop "$@"
```

The kit's program names and the `SWOOP_*` variables are the same under
every tool. They are this contract.

## swoop-match

The launcher does not filter a view's rows; the view does. `swoop-match`
is the kit's filter, for a view that prints every row and lets the kit
choose:

```sh
all_rows | swoop-match "$text"
```

- Rows on stdin, the matching rows on stdout, the best first
- A row matches when every word of the text matches its title or its
  subtitle, in any order and any case. A word matches when its letters
  are in the text in that order: `akey` matches `API key`
- A row of kind `group` stays only when a row under it matches, and the
  rows under it are ranked under it
- An empty text passes every row, in the order given
- A line that is not a row is dropped
- A row the view makes from the text, an Add row, is not for the filter:
  print it yourself, before or after the pipe

## swoop-check

`swoop-check` is the conformance test. It runs an extension the way the
launcher does and prints each thing it finds on one line, with the rule
it breaks:

```
$ swoop-check ~/.config/swoop/extensions/hello
hello: line: list: line 2 has 4 fields, not 5
```

It exits 0 when the extension meets the contract, and then prints
`hello: meets contract 1`. It takes a folder, or the program in it, and
any number of them.

It asks only the verbs that read: `list`, `view`, `preview`, `actions`,
on the first row of each kind. It does not run a row or send a text,
because it cannot know what a run does: a row may put the machine to
sleep. `-act` tries `run` and `send` too, for real, on the first row it
finds; use it on an extension whose run is safe to do once. `-limit`
changes the two seconds, for a test on a busy machine.

| rule | what it asks |
|---|---|
| `program` | the folder holds a program of the folder's name that can be run |
| `list` | `list` exits 0, with a text and without |
| `view` | `view <id>` exits 0, with a text and without, for every row of kind `view` and every view named in `key` and `views` |
| `preview` | `preview <id>` exits 0 for an id the program printed |
| `run` | `run <id>` exits 0 for an id the program printed. With `-act` only |
| `actions` | a row that `actions` prints is of kind `action`, `refresh` or `terminal` |
| `send` | a view with `bar=prompt` prints a row, and `send` exits 0 for it. The second half with `-act` only |
| `time` | `list`, `view`, `actions` and `send` are done in two seconds and leave nothing running that holds their output |
| `line` | every line `list`, `view` and `actions` print is five fields of UTF-8 with tabs between, the id not empty, under 1 MiB. An id or a title with a tab in it breaks this rule |
| `unique` | no two rows of one list have the same id |
| `group` | a header has a row under it |
| `keyword` | the `keyword` file's first line is one word |
| `key` | every line of `key` is a key that may be claimed, once, and a view |
| `views` | every line of `views` is a view, once, and settings this contract has |
| `cache` | the `cache` file holds the word `run` |
| `icons` | the `icons` file holds the word `id`, beside a `cache` file, and every id `list` prints is the path of a file |

`make test` runs it on every extension this repository bundles.

## What is not in the contract

These exist, and an extension must not lean on them. They may change in
any version.

- The launcher's other variables: `SWOOP_STATE`, `SWOOP_ONCE`,
  `SWOOP_ICONS`, `SWOOP_KIND`, `SWOOP_TITLE`, `SWOOP_SHELL`,
  `SWOOP_SHELL_PID`
- What `swoop-nav` takes, but for `rows` in a `reload` posted to fzf
- The fzf options the launcher starts with, and any fzf action but the
  two under [`views`](#views)
- The programs in `bin/` that this document does not name. They are the
  bundled extensions' own
- Where a bundled extension keeps its data

## Changes

- **1**, 2026-10-02. The first version of this document: the contract
  as [issue #2](https://github.com/beatzball/swoop/issues/2) held it on
  that day, written as one text. The issue stays as the history of how
  each part was added, and why
