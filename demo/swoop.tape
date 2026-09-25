# The launcher, recorded. Run from a throwaway directory so no home path can
# reach a frame: see "Recordings and screenshots" in AGENTS.md.
#
#   tmp=$(mktemp -d /tmp/swoop-demo.XXXX) && cp -R bin extensions "$tmp"/ && mkdir "$tmp/data" && cd "$tmp" && vhs /absolute/path/to/swoop/demo/swoop.tape
#
# XDG_DATA_HOME points into the temp directory so the Define view shows no
# words looked up on the recording machine.

Output swoop.gif

Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 22
Set Width 900
Set Height 520
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "export PATH=$PWD/bin:$PATH XDG_DATA_HOME=$PWD/data && clear"
Enter
Show

Type "swoop"
Sleep 500ms
Enter
Sleep 1.5s
Type "saf"
Sleep 1.5s
Ctrl+K
Sleep 1.5s
Escape
Sleep 1s
Backspace 3
Sleep 400ms
Type "2+2"
Sleep 1.5s
Backspace 3
Sleep 400ms
Type "def"
Sleep 1s
Enter
Sleep 1s
Type "swoop"
Sleep 2s
Escape
Sleep 800ms
Escape
Sleep 1s
Escape
Sleep 600ms
Escape
Sleep 800ms
