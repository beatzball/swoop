---
title: Agent Skill
description: One SKILL.md that teaches a coding agent to write a swoop extension, check it with swoop-check, reload the launcher, add a quicklink, and back up a settings file before it edits one.
sidebar:
  order: 7
---

## What it is

swoop's settings are text files, and an extension is a program that
prints lines. A coding agent can do the whole job of extending it, if it
knows the contract and the check tool. That knowledge is one file in the
repository, `skills/swoop/SKILL.md`, in the
[Agent Skills](https://github.com/vercel-labs/skills) format, which
Claude Code, Codex, opencode, Cursor and the other agents read.

The skill knows:

- where the config and the extensions live, and where the bundled ones are
- the five tab-separated fields of a row, and the verbs: `list`,
  `preview`, `run`, and the pane verbs `view`, `actions`, `send`
- how to run `swoop-check` on a new extension, and what each line it
  prints means
- how to reload the launcher and confirm the rows show, and what to look
  at when they do not
- how to add a quicklink
- the rule that it backs up a settings file before it edits one, and
  tells you where the copy is

It links [The Extension Contract](/docs/contract) and
[Writing an Extension](/docs/writing-an-extension) rather than repeating
them. It was written against contract 1, and says so at its top, so an
agent can compare that with the version at the top of the contract page.

The skill grants the agent nothing it does not already have. It reads and
writes the files you can, with your permissions, and runs the programs
you can. What your agent may touch is your agent's setting, not the
skill's.

## Install

With the [Skills CLI](https://github.com/vercel-labs/skills):

```sh
npx skills add https://github.com/beatzball/swoop --skill swoop
```

That installs it into the current project, for the agent you pick. Add
`-g` to install it for every project, and `-a claude-code` (or `codex`,
`opencode`, `cursor`) to skip the question.

The line names the GitHub repository, not swoop.sh. The Skills CLI finds
a skill by walking a git repository for `skills/<name>/SKILL.md`, and
that is where this one lives. A plain web address goes through a
different lookup that its documentation does not describe, so the
repository is the address that is promised to work.

Without the CLI, copy the folder `skills/swoop` from the repository into
your agent's skill directory: `.claude/skills/` for Claude Code,
`.agents/skills/` for the agents that share that folder, or whatever
yours reads.

## Three things to ask

With the skill installed, these work from one ask:

- *Write a swoop extension that lists my tmux sessions.* The agent writes
  `~/.config/swoop/extensions/sessions/sessions`, runs `swoop-check` on
  it until it meets the contract, and tells you to open the launcher and
  type `sessions`.
- *Add a quicklink to npm search.* The agent runs `swoop-links add` with
  `{argument}` in the link, or backs up `quicklinks.tsv` and appends a
  line.
- *Check why my extension shows no rows.* The agent runs `swoop-check`,
  runs `list` by hand, and looks at the `off =` line in your config, the
  folder name, and the files beside the program, in that order.
