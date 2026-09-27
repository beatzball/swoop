# The launcher, recorded. Run from a throwaway directory so no home path can
# reach a frame: see "Recordings and screenshots" in AGENTS.md.
#
#   tmp=$(mktemp -d /tmp/swoop-demo.XXXX) && cp -R bin extensions "$tmp"/ && mkdir -p "$tmp/data" "$tmp/state" "$tmp/config/swoop" && cp demo/answer "$tmp/" && printf 'sh %s/answer\n' "$tmp" > "$tmp/config/swoop/ai" && cd "$tmp" && vhs /absolute/path/to/swoop/demo/swoop.tape
#
# XDG_* point into the temp directory, so no conversation, word or setting
# from the recording machine is in a frame. The AI is demo/answer, a script
# that prints a fixed answer a line at a time: the recording is the same
# every time and no model is called.

Output swoop.gif

Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 21
Set Width 1000
Set Height 560
Set Padding 20
Set TypingSpeed 55ms

Hide
Type "export PATH=$PWD/bin:$PATH XDG_DATA_HOME=$PWD/data XDG_STATE_HOME=$PWD/state XDG_CONFIG_HOME=$PWD/config SWOOP_NO_CLIPD=1 && clear"
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
Sleep 600ms
Backspace 3
Sleep 300ms
Type "2+2"
Sleep 1.2s
Backspace 3
Sleep 400ms
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
