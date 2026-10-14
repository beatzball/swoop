# The launcher, recorded. Run from a throwaway directory so no home path can
# reach a frame: see "Recordings and screenshots" in AGENTS.md.
#
#   tmp=$(mktemp -d /tmp/swoop-demo.XXXX) && cp -R bin extensions "$tmp"/ && mkdir -p "$tmp/data/swoop" "$tmp/state" "$tmp/config/swoop" && cp demo/answer "$tmp/" && cp demo/tasks.md "$tmp/data/swoop/" && printf 'sh %s/answer\n' "$tmp" > "$tmp/config/swoop/ai" && cd "$tmp" && vhs /absolute/path/to/swoop/demo/swoop.tape
#
# XDG_* point into the temp directory, so no conversation, word or setting
# from the recording machine is in a frame. The apps are the machine's own,
# read from /Applications: a name there is not personal. The tasks are
# demo/tasks.md, and SWOOP_TASKS_NOW pins the day, so the same tasks fall
# under Past due, Today and Later on any day the tape is run. The AI is
# demo/answer, a script that prints a fixed answer a line at a time: the
# recording is the same every time and no model is called.

Output swoop.gif

Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 21
Set Width 1000
Set Height 560
Set Padding 20
Set TypingSpeed 55ms

Hide
Type "export PATH=$PWD/bin:$PATH XDG_DATA_HOME=$PWD/data XDG_STATE_HOME=$PWD/state XDG_CONFIG_HOME=$PWD/config SWOOP_NO_CLIPD=1 SWOOP_TASKS_NOW=2026-10-14T10:00 && clear"
Enter
# Nobody sees this. The programs were copied a moment ago, and the system
# checks each one the first time it runs: one to five seconds each, one
# after another, about a minute for all of them. Run every one once here,
# with nothing on stdin so each exits on its own, then one launch for the
# rest. The launch that is recorded is then as quick as an installed one.
# The word waited for is split in the typed line, so the wait matches the
# output and not the command.
Type 'for b in bin/swoop-*; do $b </dev/null >/dev/null 2>&1; done; echo rea""dy'
Enter
Wait+Screen@120s /ready/
Type "swoop"
Enter
Sleep 6s
Escape
Sleep 1s
Type "clear"
Enter
Show

Type "swoop"
Sleep 400ms
Enter
Sleep 1.2s
Type "saf"
Sleep 1.2s
Ctrl+K
Sleep 1.4s
Escape
Sleep 500ms
Backspace 3
Sleep 300ms
Type "2+2"
Sleep 1.2s
Backspace 3
Sleep 300ms
Type "task"
Sleep 700ms
Enter
Sleep 1.8s
Escape
Sleep 500ms
Backspace 4
Sleep 300ms
Tab
Sleep 900ms
Type "why is the sky blue"
Sleep 500ms
Enter
Sleep 5s
Escape
Sleep 600ms
Escape
Sleep 800ms
Escape
Sleep 600ms
