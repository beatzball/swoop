# The spike, recorded. Run from a throwaway directory so no home path can
# reach a frame: see "Recordings and screenshots" in AGENTS.md.
#
#   tmp=$(mktemp -d /tmp/swoop-demo.XXXX) && cp bin/* "$tmp"/ && cd "$tmp" && vhs /absolute/path/to/swoop/demo/spike.tape

Output spike.gif

Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 22
Set Width 900
Set Height 520
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "export PATH=$PWD:$PATH && clear"
Enter
Show

Type "swoop"
Sleep 500ms
Enter
Sleep 1s
Type "saf"
Sleep 1.5s
Backspace 3
Sleep 600ms
Type "term"
Sleep 1.5s
Escape
Sleep 800ms
