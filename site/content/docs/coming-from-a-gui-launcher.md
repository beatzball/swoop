---
title: Coming from a GUI Launcher
description: What swoop has of the things a GUI launcher taught you to expect, what it calls each one, where it lives on disk, and what swoop does not do.
sidebar:
  order: 2
---

## The same things, by swoop's names

You arrive with a launcher's words. Here is what each one is in swoop,
where it lives on disk, and the page that covers it. Every row below is an
[extension](/docs/extensions), and any extension can be turned off in
[Settings](/docs/settings).

| you expect | swoop's word | on disk | read |
|---|---|---|---|
| quicklinks | a quicklink: a name, a link, what opens it. `{argument}` in the link asks first | one line in `~/.config/swoop/quicklinks.tsv` | [Quicklinks](/docs/extensions#quicklinks) |
| snippets | a snippet, pasted into the app you came from. `{date}`, `{clipboard}`, `{cursor}` are filled | one markdown file each in `~/.config/swoop/snippets/` | [Snippets](/docs/extensions#snippets) |
| script commands | an extension: a program that prints rows, in any language that starts fast | a folder, `~/.config/swoop/extensions/<name>/<name>` | [Writing an Extension](/docs/writing-an-extension) |
| a store | none. An extension is a folder: copy it in, and it is installed | `~/.config/swoop/extensions/` | [Extensions](/docs/extensions) |
| AI chat | Ask AI: Tab from anywhere, any command that reads a question and prints an answer | the command in `~/.config/swoop/ai`, or `ai =` in `~/.config/swoop/config` | [Ask AI](/docs/ask-ai) |
| clipboard history | Clipboard History, kept by `swoop-clipd`. It skips the password managers | `~/.local/share/swoop/clipboard/history.jsonl`, yours only | [Clipboard History](/docs/extensions#clipboard-history) |
| a hotkey per command | a key, claimed by an extension. Tab is Ask AI's, alt-, is Settings'. Every other extension has a keyword instead: `clip` and a space | a file called `key` beside the extension's program. `swoop status` lists who has which | [A keyword, a key](/docs/writing-an-extension#a-keyword-a-key-a-prompt-a-preview) |
| file search | Search Files: names from Spotlight on a Mac, the preview shows the file | nothing of yours | [Search Files](/docs/extensions#search-files) |
| window moves | Window management: halves, thirds, quarters, maximize, the next display | nothing of yours. Needs Accessibility on a Mac | [Window management](/docs/extensions#window-management) |
| calculator | type the arithmetic at the root, `2+2`, and the answer is a row | nothing of yours | [Calculator](/docs/extensions#calculator) |
| emoji | `emoji`, Enter, then a name: every emoji, flag and symbol, pasted | your own words in `~/.config/swoop/emoji.keywords` | [Emoji and symbols](/docs/extensions#emoji-and-symbols) |

Two things in that table are files you may already have. A JSON export of
quicklinks (name, link, openWith) or of snippets (name, text, keyword)
comes over in one command each:

```sh
swoop-links import ~/Downloads/quicklinks.json
swoop-snippets import ~/Downloads/snippets.json
```

The hotkey that opens the panel is the one setting a GUI launcher user looks
for first. It is the Hotkey row in [Settings](/docs/settings), or `hotkey =`
in `~/.config/swoop/config`; the frame picks the change up within a second.

## What swoop does not do

Said plainly, as of 0.15.1:

- **No snippet expansion outside the panel.** Typing a snippet's keyword in
  another app does nothing. A snippet is pasted from the panel: open it,
  find the snippet, Enter.
- **No import of another launcher's settings.** Quicklinks and snippets come
  over from a JSON export, as above. Nothing else does: hotkeys, extensions
  and preferences are set again, by hand, in a text file.
- **No settings window.** Settings is a pane in the panel, one row per
  setting, and a plain text file, `~/.config/swoop/config`. There is no
  window with tabs and checkboxes.
- **The panel is a terminal, and looks like one.** It is text, drawn by
  libghostty, with icons and pictures where the terminal can draw them.
  There are no rounded cards and no mouse-first layout.
- **Releases are not signed yet.** macOS takes each upgraded frame for a new
  program: after an upgrade, remove swoop-shell-mac from Accessibility and
  add it again. [Getting Started](/docs/getting-started#start-stop-status)
  has the steps.
- **No frame on Linux or Windows yet.** swoop runs there as a terminal
  program, in the terminal you give it. On Linux, Enter on an emoji or a
  snippet copies rather than pastes.

## Move one thing at a time

Keep the old launcher running. Give swoop a hotkey the other one does not
use, and let the two sit side by side for a week.

Bring quicklinks over first. They are the smallest thing and the one you
reach for most: one import, or a few lines typed into
`~/.config/swoop/quicklinks.tsv`, and `wiki`, Enter, a term, Enter works
the way you are used to.

Then the one script you miss. Not all of them: the one. Write it as an
extension, a shell script that prints one line per result, and try it in
the launcher at each step. [Writing an Extension](/docs/writing-an-extension)
walks the whole way, and `extensions/system/system` in the repository, the
one that puts Sleep and Lock Screen in the list, is a finished one in a
screen of shell.

When those two feel like home, the rest of the table is already there.
