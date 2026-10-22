---
name: swoop
description: Extend the swoop launcher. Write an extension (a program that prints tab-separated rows), check it with swoop-check, reload the launcher and confirm the rows show, add a quicklink, and edit a settings file with a backup first. Use when asked to add, fix or check a swoop extension, quicklink or setting.
---

# swoop

swoop is a keyboard launcher. Everything in its list comes from an
extension: a small program that prints one row per line. Its settings are
plain text files. So the whole job of extending it is writing a file and
running a check, which is what this skill teaches.

Written against **contract 1** (2026-10-02). The contract is the source
of truth; this skill is a map to it. Before you write anything, read the
version at the top of https://swoop.sh/docs/contract. Same major version:
this skill holds. A different major: read the contract, and trust it over
this file where they differ.

Read, do not repeat:

- https://swoop.sh/docs/contract: every verb, kind, file and variable,
  and the rules `swoop-check` tests
- https://swoop.sh/docs/writing-an-extension: one extension built a step
  at a time, from three rows to a pane with actions
- https://swoop.sh/docs/settings: every setting and every file
- https://swoop.sh/docs/extensions: what ships, including quicklinks

This skill grants nothing. The agent reads and writes the same files the
user can, with the same permissions, and runs the same programs.

## Where things live

| what | where |
|---|---|
| the user's settings | `~/.config/swoop/config`, one `key = value` per line |
| the user's extensions | `~/.config/swoop/extensions/<name>/<name>` |
| quicklinks | `~/.config/swoop/quicklinks.tsv`: name, link, app, tab-separated |
| the bundled extensions, installed with curl | `~/.local/share/swoop/current/extensions/` |
| the bundled extensions, installed with Homebrew | `$(brew --prefix)/opt/swoop/libexec/extensions/` |
| the bundled extensions, from a checkout | `extensions/` in the repository |

`$XDG_CONFIG_HOME` stands in for `~/.config` when it is set. An extension
of the user's with the same name as a bundled one replaces it. `swoop
status` says whether swoop runs, where from, and how it was installed.

The kit's programs (`swoop-check`, `swoop-links`, `swoop-match`) sit
beside `swoop`. Find the check tool once, then use what this prints:

```sh
command -v swoop-check \
  || ls ~/.local/share/swoop/current/bin/swoop-check \
  || ls "$(brew --prefix)/opt/swoop/libexec/bin/swoop-check"
```

From a checkout, `make build` puts it in `bin/swoop-check`.

## Back up a settings file before you edit it

Every settings file is the user's. Before you change one, copy it beside
itself with the date in the name, and tell the user the copy's path:

```sh
f=~/.config/swoop/config
[ -f "$f" ] && cp "$f" "$f.bak-$(date +%Y%m%d-%H%M%S)"
```

The same for `quicklinks.tsv`, `ai`, `emoji.keywords`, `clipboard.ignore`,
`shell-mac.json` and anything else under `~/.config/swoop/`. Then edit
the file in place, one line per setting, and keep every line you did not
mean to change. Write the new file beside the old and move it over, so a
reader never sees half a file. The frame watches `config`, so a change
works within a second, with no restart.

An extension turned off is one line in `config`:

```
off = reminders, tasks
```

If the user's new extension shows no rows, check that line first.

## The row

An extension prints result lines: one row per line, five fields, tabs
between them.

```
id	kind	icon	title	subtitle
```

- `id` is yours; swoop hands it back in `preview` and `run`. Never empty,
  never shown
- `kind` says what Enter does. A word of your own means run it. The
  special kinds are `view` (opens a pane filled by your `view` verb),
  `action` and `refresh` (rows of a ctrl-k menu), `toggle`, `terminal`
  and `group` (a header). The contract has each one
- `icon` is one emoji or glyph, or empty
- `title` is what the user reads and what typing matches
- `subtitle` is dimmer text after it, or empty

No tab inside a field. Every row of one list has its own id.

## The verbs

The launcher runs the program once per verb, in the extension's own
folder, with the kit's programs on `PATH`:

| verb | when | prints |
|---|---|---|
| `list [text]` | the launcher opens, or on each keystroke | rows for the main list |
| `preview <id>` | on every cursor move | the right-hand pane, any text |
| `run <id> [action]` | Enter, or an action picked from ctrl-k | nothing; does the thing and exits |
| `view <id> [text]` | a row of kind `view` was opened; again on each keystroke | the pane's rows, already filtered |
| `actions <id>` | ctrl-k on a row | rows whose id is an action's name |
| `send <id> <text>` | Enter in a view with `bar=prompt` | nothing |

A verb the extension does not know exits non-zero with one line on
stderr. `list`, `view` and `actions` have two seconds; `preview` runs on
every cursor move, so keep it quick. Three small files beside the program
change how it is asked: `keyword`, `key`, `cache`. The contract has them.

## Write an extension

1. Pick a name: lower case, one word, not a bundled one unless the user
   wants to replace it. The folder and the program share it.
2. Make the folder and the program:

   ```sh
   name=sessions
   mkdir -p ~/.config/swoop/extensions/$name
   ```

3. Start from this shape. It is bash, but any language works:

   ```bash
   #!/usr/bin/env bash
   # sessions: one row per tmux session. Enter switches to it.
   set -euo pipefail

   # One result line: id, kind, icon, title, subtitle, tab-separated.
   row() { printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5"; }

   case "${1:-}" in
     list)
       tmux list-sessions -F '#{session_name}	#{session_windows}' 2>/dev/null |
         while IFS=$'\t' read -r s n; do
           row "$s" session '🖥' "$s" "$n windows"
         done
       ;;
     preview)
       tmux list-windows -t "$2" -F '#{window_index}: #{window_name}' 2>/dev/null
       ;;
     run)
       tmux switch-client -t "$2" 2>/dev/null || tmux attach -t "$2"
       ;;
     *)
       echo "usage: sessions list | preview <id> | run <id>" >&2
       exit 2
       ;;
   esac
   ```

   Keep the `*)` branch: the launcher may ask a verb you did not write,
   and the right answer is a non-zero exit. `list` with nothing to show
   prints nothing and exits 0.

4. Make it a program, then run it by hand before the launcher does:

   ```sh
   chmod +x ~/.config/swoop/extensions/$name/$name
   ~/.config/swoop/extensions/$name/$name list | cat -vet | head
   ```

   Each line shows four `^I` (a tab) between five fields, and ends in `$`.

5. Check it (next section). Then reload and verify (the section after).

For a pane of its own, actions, a keyword or a key, follow the steps in
https://swoop.sh/docs/writing-an-extension; each ends with a check.

## Check it with swoop-check

```sh
swoop-check ~/.config/swoop/extensions/$name
```

It runs the extension the way the launcher does and prints one line per
rule it breaks, as `name: rule: verb: what`. Exit 0 and
`name: meets contract 1` means the extension meets the contract. The
`(not tried without -act: run)` after it is normal: it asks only the
verbs that read, and never runs a row. Add `-act` to try `run` on the
first row too, only when that run is safe to do once.

What a line means, and what to do:

| it says | do |
|---|---|
| `program: ... is not there` | the folder has no runnable program of its own name: check the name, `chmod +x`, the `#!` line |
| `line: list: line 1 has no tab in it` | the row was printed with `echo` or spaces. Print with `printf '%s\t%s\t%s\t%s\t%s\n'` |
| `line: list: line 1 has 4 fields, not 5` | a tab is missing, or a tab sits inside an id or a title and makes one more |
| `line: list: line 1 has an empty id` | the first field was blank on that row |
| `unique: list: rows 1 and 2 have the same id` | give every row of a list its own id |
| `list: exit status 1` or `preview: ... exit status 2` | run that verb by hand with the same arguments and read stderr |
| `time: list took longer than 2s` | too slow. Cache the slow part, or put the word `run` in a file called `cache` beside the program, so `list` runs once per launch with ten seconds |
| `actions: ...` | an actions row must be of kind `action`, `refresh` or `terminal` |
| `keyword`, `key`, `views`, `cache`, `icons` | that file beside the program does not hold what the contract says; the line names the word it wanted |

Fix, run it again, repeat until it meets the contract.

## Reload and verify

The launcher reads the extensions folder every time it opens. There is
nothing to register and nothing to restart.

1. In the frame: press the hotkey to show the panel, Esc to hide it, the
   hotkey again. The new rows are in the list.
2. In a terminal: run `swoop`, type the row's title, and look. Esc quits.
3. `swoop status` names any extension whose `list` was too slow the last
   time the launcher was slow to open.

Then, in the launcher, move the cursor onto a row to see the preview, and
press Enter once on a row whose run is safe.

If the rows are not there, in this order:

- `swoop-check` passes? If not, fix that first
- the program prints rows when you run `list` by hand, as the user, from
  the extension's folder?
- the folder name and the program name are the same word?
- `off =` in `~/.config/swoop/config` names it?
- a bundled extension has the same name? Yours wins, so the bundled rows
  are gone, which may be the surprise
- a `keyword` file beside it? Then its rows show after that word and a
  space, not in the main list
- a `cache` file beside it? Then a change shows the next time the
  launcher opens, not on the next keystroke
- `swoop status` says swoop runs from a different install than the one
  you checked with?

## Add a quicklink

A quicklink is a name, a link, and what opens it. The `links` extension
reads `~/.config/swoop/quicklinks.tsv`. Let the tool write it:

```sh
swoop-links add 'npm' 'https://www.npmjs.com/search?q={argument}'
swoop-links add 'Home' ~ Finder
```

`{argument}` in the link makes it ask first: Enter opens a pane, what the
user types goes into the link, encoded. To edit the file by hand, back it
up first, then append one line: name, link, app, tab-separated. The row
shows the next time the launcher opens.

## Rules of the road

- Back up before you edit a settings file, and say where the copy is
- Never print a row with `echo`: use `printf` with `\t` between fields
- Keep `list` and `preview` fast. Cache what is slow
- A verb you do not handle exits non-zero with one line on stderr
- Do not lean on anything the contract lists under What is not in the
  contract
- Run `swoop-check` after every change, and leave the extension meeting
  the contract
