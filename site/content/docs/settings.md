---
title: Settings
description: One row per setting, the value beside it, Enter to change it. Every value is one line in a plain text file you can also edit.
sidebar:
  order: 4
---

## Open Settings

Type `settings` and press Enter, or press cmd+, in the frame (alt+, in a
terminal). One row per setting, the current value beside it. Enter on a row
to change it.

## What you can set

| setting | in the file | what it changes |
|---|---|---|
| Hotkey | `hotkey` | the key that shows the panel |
| Preview width | `preview` | where the divider between the list and the preview sits, in percent |
| AI model | `ai` | which command answers in [Ask AI](/docs/ask-ai) |
| Web search for AI | `web` | whether that model may search the web |
| API URL | `ai_url` | the server an `openai:` or `lmstudio:` model talks to |
| API key | `ai_key` | the key for that server; empty uses `OPENAI_API_KEY` |
| Editor | `editor` | what notes, tasks and Edit actions open in, inside the panel |
| Used recently | `recent` | whether the five things you opened last come first |
| Week starts on | `week` | `monday` or `sunday`: where This week ends in Tasks and Reminders |
| Emoji skin tone | `skin` | the default tone for emoji |
| Transcript renderer | `render` | what draws the Ask AI transcript |
| Extensions | `off` | which extensions are off |

The last row, Open the config folder, shows you the files.

## Turn an extension off

Extensions lists every extension with a box for on or off. Enter turns one
off, and its rows leave the root list until you turn it back on. Settings
itself cannot be turned off.

In the file that is one line:

```
off = reminders, tasks
```

## The file

Every value is one line in `~/.config/swoop/config`, which you can also edit
by hand:

```
hotkey = alt+shift+space
preview = 58
editor = nvim
ai = claude
render = glow -s dark
off = reminders
```

The frame watches that file, so a new hotkey works within a second, with no
restart.

## Other files

| file | what it holds |
|---|---|
| `~/.config/swoop/ai` | a custom AI command, one line |
| `~/.config/swoop/quicklinks.tsv` | your quicklinks |
| `~/.config/swoop/snippets/` | your snippets, one markdown file each |
| `~/.config/swoop/emoji.keywords` | your own words for emoji |
| `~/.config/swoop/clipboard.ignore` | apps whose copies are never kept |
| `~/.config/swoop/shell-mac.json` | the panel's size and font size |
| `~/.config/swoop/extensions/` | your own [extensions](/docs/writing-an-extension) |

`XDG_CONFIG_HOME` replaces `~/.config` when it is set.
