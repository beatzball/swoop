---
title: Writing an Extension
description: Build a swoop extension from nothing, a shell script that lists, previews and runs, then opens a pane, then offers actions, and try it in the launcher at every step.
sidebar:
  order: 5
---

## What you will build

`clock`: the time in the places you care about. By the end it has:

- a row per city with the time now, and the date in the preview
- Enter to copy the time
- a Find a Time Zone pane that searches every zone the system knows
- a ctrl-k menu to copy the time another way, and to add or remove a city
- a city picker that stays open while you tick, and an Edit action that opens
  your editor inside the panel

It is one bash script, and every step is a working extension. Each step ends
with a check you can run.

The contract itself, every verb and every row kind, is
[issue #2](https://github.com/beatzball/swoop/issues/2). It is the source of
truth. This page teaches it; where they differ, the issue wins.

## Before you start

You need swoop, and a way to run the launcher in a terminal. From a checkout:

```sh
make build
bin/swoop     # Esc quits
```

If you installed swoop with Homebrew or curl, `swoop` does the same, and the
hotkey panel works too. Examples below say `bin/swoop`.

An extension is a folder with one program in it, named the same:

```sh
mkdir -p ~/.config/swoop/extensions/clock
touch ~/.config/swoop/extensions/clock/clock
chmod +x ~/.config/swoop/extensions/clock/clock
```

swoop finds it the next time the launcher opens. No restart, no
registration, no manifest. The folder name is the extension's name.

## The result line

Everything an extension prints is result lines: one row per line, five
fields, separated by tabs.

```
id	kind	icon	title	subtitle
```

| field | what it is |
|---|---|
| `id` | yours. swoop hands it back to you in `preview` and `run`. Never shown |
| `kind` | what Enter does. A word of your own means "run it"; the special kinds are below. Never shown |
| `icon` | one emoji or Nerd Font glyph. May be empty |
| `title` | the main text. Typing matches on this |
| `subtitle` | dimmer text after the title. May be empty |

The special kinds, each one met in a step below:

| kind | Enter does |
|---|---|
| `view` | opens a pane filled by your `view` verb |
| `action` | in a ctrl-k menu: runs the action and closes the launcher |
| `refresh` | in a ctrl-k menu: runs it and comes back to the pane, reloaded |
| `toggle` | runs it and stays in the same pane, reloaded |
| `terminal` | runs it with the whole terminal, then comes back, reloaded |
| `group` | nothing: the row is a header over the rows below it |

A `group` row is not in a step. Tasks prints one over each run of tasks:
Today, Tomorrow. Give it an id like any row, and print it only when it has
rows under it. The launcher draws its title at the left edge, dimmed, over
the icons of the rows below, and the cursor never stops on it: Up and Down
step over it, and a click on it does nothing. Leave its icon empty.

## Step 1: list

`list` prints the rows that join the launcher's main list. Put this in
`~/.config/swoop/extensions/clock/clock`:

```bash
#!/usr/bin/env bash
# clock: the time in a few places, one row each.
set -euo pipefail

zones='Europe/London America/New_York Asia/Tokyo'

# One result line: id, kind, icon, title, subtitle, tab-separated.
row() { printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5"; }

# Europe/London -> London, America/New_York -> New York.
city() { printf '%s' "${1##*/}" | tr _ ' '; }

case "${1:-}" in
  list)
    for z in $zones; do
      row "$z" clock '🕐' "Time in $(city "$z")" "$(TZ=$z date +%H:%M)"
    done
    ;;
  *)
    echo "usage: clock list" >&2
    exit 2
    ;;
esac
```

`zones` is split on spaces by the `for`, which is why it is not quoted
there. The `*)` branch matters: swoop may call a verb you have not written
yet, and the right answer is a non-zero exit.

**Try it.** First on its own, where a mistake is easy to see:

```sh
~/.config/swoop/extensions/clock/clock list
```

Three lines, each with four tabs. Then in the launcher:

```sh
bin/swoop
```

Type `time in`. The three rows are there, the time beside each. Do not press
Enter yet: there is no `run`. Esc quits.

## Step 2: preview

`preview <id>` prints the right-hand pane for one row. It runs on every
cursor move, so it must be quick. Add a branch to the `case`:

```bash
  preview)
    # $2 is the id from the row: the zone.
    TZ="$2" date '+%A %e %B%n%n%H:%M:%S%n%n%Z, UTC%z'
    ;;
```

The id you printed in `list` is the id you get back, as it was. swoop adds
its own prefix, `ext/clock/`, to route the row, and takes it off again
before it calls you.

**Try it.**

```sh
~/.config/swoop/extensions/clock/clock preview Asia/Tokyo
bin/swoop
```

Type `time in tokyo`. The preview shows the day, the time to the second, and
`JST, UTC+0900`. Move between the rows and it follows.

The preview may print anything a terminal draws: colors as ANSI escapes, and
pictures through `swoop-img`. `FZF_PREVIEW_COLUMNS` and `FZF_PREVIEW_LINES`
say how big the pane is.

## Step 3: run

`run <id>` does the thing, then exits. Enter calls it and closes the
launcher. Add a clipboard helper above the `case`, and a `run` branch:

```bash
# Whichever clipboard tool this system has.
copy() {
  if command -v pbcopy >/dev/null; then pbcopy
  elif command -v wl-copy >/dev/null; then wl-copy
  else xclip -selection clipboard
  fi
}
```

```bash
  run)
    TZ="$2" date +%H:%M | tr -d '\n' | copy
    ;;
```

That is a complete extension: `list`, `preview`, `run`. The `system`
extension in the repository, which puts Sleep and Lock Screen in the list, is
exactly this shape.

**Try it.**

```sh
~/.config/swoop/extensions/clock/clock run Asia/Tokyo && pbpaste   # wl-paste on Linux
bin/swoop
```

Type `time in tokyo`, press Enter. The launcher closes, and the time in Tokyo
is on your clipboard.

A `run` that fails should exit non-zero, with one line on stderr that says
why.

## Step 4: view

Three cities is a start. To choose from all of them, give the extension a
pane of its own. A row of kind `view` does not run on Enter: it opens a
pane, and swoop asks your `view` verb for the rows, again on every keystroke,
with what is in the bar.

Add a helper that lists every zone, a `view` row to `list`, the `view`
verb, and a preview for the new row:

```bash
# Every zone the system knows, one per line, sorted.
all_zones() { awk -F'\t' '!/^#/ { print $3 }' /usr/share/zoneinfo/zone.tab | sort; }
```

```bash
  list)
    for z in $zones; do
      row "$z" clock '🕐' "Time in $(city "$z")" "$(TZ=$z date +%H:%M)"
    done
    row find view '🌐' 'Find a Time Zone' 'Every zone, by city'
    ;;
  view)
    # $2 is the view row's id ("find"), $3 the text in the bar. The
    # launcher shows exactly these rows, in this order: filtering is ours.
    all_zones | grep -i -F -- "${3:-}" | head -100 | while IFS= read -r z; do
      row "$z" clock '🕐' "$(city "$z")" "$z"
    done
    ;;
  preview)
    case "$2" in
      find) echo 'Type a city to narrow the list.' ;;
      *) TZ="$2" date '+%A %e %B%n%n%H:%M:%S%n%n%Z, UTC%z' ;;
    esac
    ;;
```

In the main list, swoop filters your rows for you. In a view it does not:
you get the text, and it shows what you print. That is what lets a view
search something bigger than a list, like a dictionary or the disk. It also
means the view runs on every keystroke, so keep it fast, and cap what it
prints: here `head -100`.

The rows of a view are ordinary rows. `preview` and `run` work on them as
anywhere else, which is why Enter on Lisbon already copies the time.

`grep` finds the text exactly as typed. To filter the way the bundled
views do, print every row and pipe the rows through `swoop-match`, a small
tool that ships with swoop:

```bash
  view)
    all_zones | while IFS= read -r z; do
      row "$z" clock '🕐' "$(city "$z")" "$z"
    done | swoop-match "${3:-}" | head -100
    ;;
```

`swoop-match <query>` reads rows and prints the ones that match, the best
first:

- every word typed must match the title or the subtitle, in any order and
  any case
- a word matches when its letters are there in that order, so `lsbn` finds
  Lisbon; the letters must sit close together, so a long text does not
  match every word
- a word as typed comes ahead of one with letters in between, the start of
  a word ahead of its middle, the title ahead of the subtitle
- a header, a row of kind `group`, stays only when a row under it matches
- with nothing typed, every row passes, in your order

A row you make from the text itself, like an Add row, is yours to print:
put it before or after the pipe. The `reminders` extension does this.

**Try it.**

```sh
~/.config/swoop/extensions/clock/clock view find lisb
bin/swoop
```

Type `find a time`, Enter. The bar now says `Find a Time Zone >`. Type
`lisb`: one row, Lisbon, with its time in the preview. Esc clears the bar;
Esc again goes back to the main list, with your text.

## Step 5: actions

ctrl-k on a row opens a menu of what else it can do. The `actions <id>` verb
fills that menu, with result lines. In an action row the id is the action's
name, and swoop calls `run <id> <action>` when it is picked. The kind says
what happens after:

- `action` runs it and closes the launcher
- `refresh` runs it and comes back to the pane the row was in, reloaded

With actions, the three cities can become a list you keep. Replace the
`zones=` line with a file, and helpers to read and change it:

```bash
list_file="${XDG_CONFIG_HOME:-$HOME/.config}/swoop/clock.zones"

# Your zones: the file if there is one, else three to start with.
zones() {
  if [ -f "$list_file" ]; then cat "$list_file"
  else printf '%s\n' Europe/London America/New_York Asia/Tokyo
  fi
}
on_list() { zones | grep -q -x -F -- "$1"; }
# stdin becomes the list. Written beside it and moved, so a reader never
# sees half a file.
save() { mkdir -p "$(dirname "$list_file")"; cat > "$list_file.new"; mv "$list_file.new" "$list_file"; }
```

`list` reads the file now:

```bash
  list)
    zones | while IFS= read -r z; do
      row "$z" clock '🕐' "Time in $(city "$z")" "$(TZ=$z date +%H:%M)"
    done
    row find view '🌐' 'Find a Time Zone' 'Every zone, by city'
    ;;
```

Then the menu, and a `run` that knows the actions:

```bash
  actions)
    # What ctrl-k offers on a row: result lines whose id is the action's
    # name. "action" runs it and closes; "refresh" runs it and comes back.
    [ "$2" != find ] || exit 0
    row time action '' 'Copy the time' "$(TZ="$2" date +%H:%M)"
    row iso action '' 'Copy as ISO 8601' "$(TZ="$2" date +%Y-%m-%dT%H:%M:%S%z)"
    if on_list "$2"; then
      row remove refresh '' 'Remove from your list' ''
    else
      row add refresh '' 'Add to your list' ''
    fi
    ;;
  run)
    # $3 is the action from the menu, or nothing for Enter.
    case "${3:-time}" in
      time) TZ="$2" date +%H:%M | tr -d '\n' | copy ;;
      iso) TZ="$2" date +%Y-%m-%dT%H:%M:%S%z | tr -d '\n' | copy ;;
      add) { zones; echo "$2"; } | save ;;
      remove) { zones | grep -v -x -F -- "$2" || true; } | save ;;
      *) echo "clock: unknown action: $3" >&2; exit 2 ;;
    esac
    ;;
```

An extension without `actions` simply has no menu. Printing nothing, as this
one does for the Find row, is the same.

**Try it.**

```sh
~/.config/swoop/extensions/clock/clock actions Europe/Lisbon
bin/swoop
```

Type `find a time`, Enter, `lisb`. Press ctrl-k: Copy the time, Copy as ISO
8601, Add to your list. Type `add`, Enter. You are back in the pane, and
Lisbon is on your list. Quit with Esc, run `bin/swoop` again, and type
`time in`: Lisbon is there with the other three.

## Step 6: toggle and terminal

Two more kinds, for rows that should not close anything.

A `toggle` row runs on Enter and the pane stays, reloaded, with the bar
emptied and the cursor where it was. Tasks uses it to tick a task. Here it
turns the Find pane into a picker: Enter puts a city on your list, or takes
it off.

A `terminal` row gets the whole terminal while it runs, and the launcher
comes back after, reloaded, the preview redrawn. Notes uses it to open an
editor. Here it is an Edit your list action.

The Find rows need their own ids now, so that `run` can tell their Enter
(toggle) from a clock row's Enter (copy). Prefix them with `pick:`, and
strip it wherever a zone is needed. The whole script, as it ends:

```bash
#!/usr/bin/env bash
# clock: the time in the places on your list, one row each. Enter copies
# it; ctrl-k copies it another way, changes the list, or opens it in your
# editor. "Find a Time Zone" opens a pane of every zone there is, where
# Enter puts a zone on your list or takes it off.
set -euo pipefail

list_file="${XDG_CONFIG_HOME:-$HOME/.config}/swoop/clock.zones"

# One result line: id, kind, icon, title, subtitle, tab-separated.
row() { printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5"; }

# Europe/London -> London, America/New_York -> New York.
city() { printf '%s' "${1##*/}" | tr _ ' '; }

# Whichever clipboard tool this system has.
copy() {
  if command -v pbcopy >/dev/null; then pbcopy
  elif command -v wl-copy >/dev/null; then wl-copy
  else xclip -selection clipboard
  fi
}

# Every zone the system knows, one per line, sorted.
all_zones() { awk -F'\t' '!/^#/ { print $3 }' /usr/share/zoneinfo/zone.tab | sort; }

# Your zones: the file if there is one, else three to start with.
zones() {
  if [ -f "$list_file" ]; then cat "$list_file"
  else printf '%s\n' Europe/London America/New_York Asia/Tokyo
  fi
}
on_list() { zones | grep -q -x -F -- "$1"; }
# stdin becomes the list. Written beside it and moved, so a reader never
# sees half a file.
save() { mkdir -p "$(dirname "$list_file")"; cat > "$list_file.new"; mv "$list_file.new" "$list_file"; }

# A row in the Find pane has the id pick:<zone>, so run can tell its
# Enter (put it on the list, or take it off) from a clock row's (copy).
zone_of() { printf '%s' "${1#pick:}"; }

case "${1:-}" in
  list)
    zones | while IFS= read -r z; do
      row "$z" clock '🕐' "Time in $(city "$z")" "$(TZ=$z date +%H:%M)"
    done
    row find view '🌐' 'Find a Time Zone' 'Every zone, by city'
    ;;
  view)
    # $2 is the view row's id ("find"), $3 the text in the bar. The
    # launcher shows exactly these rows, in this order: filtering is ours.
    # "toggle": Enter runs the row and the pane stays, reloaded.
    all_zones | grep -i -F -- "${3:-}" | head -100 | while IFS= read -r z; do
      if on_list "$z"; then mark='✓ on your list'; else mark="$z"; fi
      row "pick:$z" toggle '🕐' "$(city "$z")" "$mark"
    done
    ;;
  preview)
    case "$2" in
      find) echo 'Type a city to narrow the list.' ;;
      *) TZ="$(zone_of "$2")" date '+%A %e %B%n%n%H:%M:%S%n%n%Z, UTC%z' ;;
    esac
    ;;
  actions)
    # What ctrl-k offers on a row: result lines whose id is the action's
    # name. "action" runs it and closes; "refresh" runs it and comes back;
    # "terminal" hands it the whole terminal, and comes back after.
    [ "$2" != find ] || exit 0
    z="$(zone_of "$2")"
    row time action '' 'Copy the time' "$(TZ="$z" date +%H:%M)"
    row iso action '' 'Copy as ISO 8601' "$(TZ="$z" date +%Y-%m-%dT%H:%M:%S%z)"
    if on_list "$z"; then
      row remove refresh '' 'Remove from your list' ''
    else
      row add refresh '' 'Add to your list' ''
    fi
    row edit terminal '' 'Edit your list' "${EDITOR:-nano}"
    ;;
  run)
    z="$(zone_of "$2")"
    action="${3:-}"
    # Enter on a Find row is the toggle; Enter anywhere else copies.
    if [ -z "$action" ]; then
      case "$2" in
        pick:*) if on_list "$z"; then action=remove; else action=add; fi ;;
        *) action='time' ;;
      esac
    fi
    case "$action" in
      time) TZ="$z" date +%H:%M | tr -d '\n' | copy ;;
      iso) TZ="$z" date +%Y-%m-%dT%H:%M:%S%z | tr -d '\n' | copy ;;
      add) { zones; echo "$z"; } | save ;;
      remove) { zones | grep -v -x -F -- "$z" || true; } | save ;;
      # Unquoted, so an EDITOR with flags, like "code -w", still works.
      edit) [ -f "$list_file" ] || zones | save; ${EDITOR:-nano} "$list_file" ;;
      *) echo "clock: unknown action: $action" >&2; exit 2 ;;
    esac
    ;;
  *)
    echo "usage: clock list | view find [query] | preview <id> | actions <id> | run <id> [action]" >&2
    exit 2
    ;;
esac
```

**Try it.**

```sh
bin/swoop
```

Type `find a time`, Enter, `lisb`, Enter. The pane stays, the bar empties.
Type `lisb` again: Lisbon says `✓ on your list`. Press ctrl-k, type `edit`,
Enter: your editor opens on the list, inside the launcher. Quit the editor,
and you are back in the pane.

## A keyword, a key, a prompt, a preview

Four more things an extension can have. None is a verb: each is a line in
a small file beside the program, which swoop reads without running
anything. Ask AI and Settings are built from these, and the launcher knows
neither by name.

```
~/.config/swoop/extensions/clock/clock      the program
~/.config/swoop/extensions/clock/keyword    time
~/.config/swoop/extensions/clock/key        alt-t find Find a Time Zone
~/.config/swoop/extensions/clock/views      find preview=45%,wrap
```

### keyword

One word. `time` and a space at the start of the bar then asks only this
extension. When the extension claims a key, the keyword opens the same
view as the key. Otherwise, when its `list` is one row of kind `view`, the
keyword opens that view, with the rest of the bar typed into it.

### key

One claim per line: the key, the id of the view it opens, and the view's
title, which becomes the pane's prompt. The key works from anywhere in the
launcher and does what Enter on the view's row would do. Inside that view
it does nothing.

| you may claim | |
|---|---|
| `tab`, `shift-tab` | |
| `f1` to `f12` | |
| `alt-` and one of `a` to `z`, `0` to `9`, `,` `.` `/` | lower case. It takes the place of what fzf does with that key |

Every other key is the launcher's or the bar's. A claim for one is left
out. When two extensions claim the same key, the first by name has it.
`swoop status` lists who has which key and every claim that lost. An
extension turned off in Settings gives its keys up, and the next that
claims one has it at once.

### views: a bar that is a prompt

A line in `views` is a view's id, then settings. `bar=prompt` makes the
bar text to send, not a filter:

```
ask bar=prompt
```

- Typing does not filter the rows, and `view` is not asked again. It gets
  no text
- Enter runs `clock send <id> <text>`: the row under the cursor and the
  bar's text. With an empty bar Enter does nothing
- Exit 0: the bar clears and the rows are listed again. To put the cursor
  on a row, write its id to the file in `$SWOOP_LAND` before you exit
- Any other exit: the text stays in the bar, so nothing typed is lost
- A key that opens this view keeps the bar's text: it is the prompt about
  to be sent

`send` has two seconds, like `list`. For an answer that takes longer,
start a worker, leave it running, and exit. The worker redraws the pane by
asking fzf: swoop starts fzf with its HTTP API on, and your program gets
the socket in `$FZF_SOCK` (a port in `$FZF_PORT` on Windows) and the key
in `$FZF_API_KEY`.

```sh
curl -s --unix-socket "$FZF_SOCK" -H "x-api-key: $FZF_API_KEY" \
  --data-binary 'reload(swoop-nav rows)+refresh-preview' http://fzf/
```

`refresh-preview` draws the preview again, and `reload(swoop-nav rows)`
lists the pane again. Post nothing else. That is how Ask AI shows an
answer as it arrives.

### views: a preview of its own

`preview=` and a list, separated by commas:

| value | the preview |
|---|---|
| `45%` | takes that much of the width, 20 to 80. Without one, the width the user chose |
| `wrap` | wraps long lines, as prose wants |
| `follow` | keeps its end in view as the text grows |

It holds while the view is open, and its actions menu keeps it. In a view
with a width of its own the divider keys do nothing.

## Make it fast

`list` runs when the launcher opens, and `preview` on every cursor move. A
view's `view` runs on every keystroke. What feels instant in a shell can be
slow sixty times a second.

- Do the slow part once and cache it, in `$XDG_CACHE_HOME` or beside the
  script: `SWOOP_EXT_DIR` is the extension's own folder
- Cap a view's rows. Nobody reads the thousandth
- `swoop-match` filters a few thousand rows in a few milliseconds. The
  cost is in making the rows, so make them once and cache them
- A long list that does not depend on the bar, and does not change while
  the launcher is open, can be listed once per launch: see below
- A script is fine. When one grows, the bundled extensions keep a short shell
  script as the entry point and put the work in a compiled tool: `emoji`,
  `notes` and `tasks` do this

## Files beside the program

A few things the launcher must know before it runs anything. Each is a
small file in the extension's folder, beside the program, with one word on
its first line.

| file | word | what it asks for |
|---|---|---|
| `keyword` | yours, `clock` | that word and a space in the bar asks only this extension. When its `list` is one row of kind `view`, the keyword opens that view |
| `cache` | `run` | `list` runs once per launch, with no text, and the rows are kept until the launcher closes |
| `icons` | `id` | each row's id is the path of a file, and the row shows that file's icon in place of the glyph. Needs `cache` |

Without `cache`, the main list asks `list` again on every keystroke, with
the text in the bar as `$2`: that is how the calculator answers `2+2`.
With it, `list` is asked once, when the launcher opens, side by side with
the other extensions, and typing only filters the rows kept. Use it for a
list that is long and fixed for the moment: the bundled `apps` extension
lists the applications this way.

```sh
echo run > ~/.config/swoop/extensions/clock/cache
```

A kept list is a list the launcher does not ask about again, so a change
shows the next time the launcher opens. With a keyword typed, the scoped
list still asks `list` with the text, each time.

`icons` is for rows that stand for a file with a picture of its own. Print
the glyph as the fallback: a terminal that draws no pictures shows it, and
so does the first launch, while the pictures are made ready in the
background. Today the launcher can take the icon of an application bundle
on a Mac; any other file keeps its glyph.

```sh
echo id > ~/.config/swoop/extensions/clock/icons
```

## Where to go next

- [Issue #2](https://github.com/beatzball/swoop/issues/2): the contract,
  every verb, kind and file, and why each one was added
- `extensions/` in the repository: sixteen working extensions. `system` is
  the smallest; `define` is the smallest with a view; `files` has a
  `terminal` action
- [Extensions](/docs/extensions): what ships, to see what each kind feels
  like
