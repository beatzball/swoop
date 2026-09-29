---
title: Ask AI
description: Press Tab and ask. swoop runs any command you name with the conversation on its stdin and shows what it prints, as it prints it.
sidebar:
  order: 2
---

## Tab, type, Enter

Press Tab from anywhere. The pane opens with whatever you had typed still in
the bar. Enter sends it, the bar clears, and the answer arrives on the
right: your prompt, dots while the model thinks, then the text as it comes.

Type again and press Enter to continue the same conversation; each exchange
appends below the last. The list under `Ask AI >` holds your conversations,
newest first. Move to one to read it. ctrl-k on it offers Copy last answer,
Copy conversation, and Delete. Esc brings the launcher back with the text you
had before Tab.

## Pick a model

swoop does not know what a model is. It runs one command with the
conversation so far on its stdin and shows what comes out of its stdout, as
it comes. Name the command in `~/.config/swoop/ai`, one line:

```
ollama run llama3.2
```

Or pick one in [Settings](/docs/settings), under AI model, or with ctrl-k on
the New conversation row in the pane:

| you have | pick | streams | searches the web |
|---|---|---|---|
| Claude Code | `claude` | yes | yes, with the switch |
| Codex | `codex` | no, answers whole | yes, with the switch |
| Ollama | `ollama:<model>`, each one listed | yes | no |
| LM Studio | `lmstudio:<model>`, each loaded one listed | yes | no |
| an OpenAI key | `openai:<model>`, key under API key | yes | yes, with the switch |
| OpenRouter, Groq, vLLM, any server with the OpenAI API | `openai:<model>`, its address under API URL | yes | no |
| anything else | the line in `~/.config/swoop/ai` | if it does | if it does |

Without a choice it uses the line in that file, then `claude -p` if `claude`
is on your PATH, then `ollama run` with the first model `ollama list` shows.

## Web search

Web search is off until you turn it on in Settings. When it is on, the
models that can will search and fetch, and the line under the dots says when
the one you picked cannot.

Anything that reads a question and prints an answer works, streaming or not.

## Where conversations live

Conversations are files in `~/.local/state/swoop/ai`, readable by you only.
Nothing is sent anywhere but to the command you named.

## How answers are drawn

Answers are markdown, drawn by swoop's own small renderer. It is also a
command, `swoop-md`, for anything else that wants markdown in a terminal:

```sh
printf '# hi\n\n- one\n- two\n' | swoop-md -w 40
```

The transcript can be drawn by any command instead. One line in
`~/.config/swoop/config`, the answer on its stdin, its stdout shown:

```
render = glow -s dark
```
