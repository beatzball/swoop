# shellcheck shell=bash
# launchd.sh: the one place that writes the tool's launchd agents. Sourced, not
# run, by both ways in: `scripts/install` (a checkout, through `make install`)
# and `scripts/get` (a downloaded release). Two copies of a plist would drift;
# this way a fix to one is a fix to both. It ships in the release tarball, so
# `scripts/get` needs nothing from a checkout.
#
#   swoop_launchd DIR FRAME
#
# DIR holds bin/swoop, bin/swoop-clipd and the file `tool`: a checkout, or
# an install such as ~/.local/share/<name>/current. FRAME is the frame's
# executable, with its resource bundle beside it. Two agents run at login
# and keep running, labelled with the tool's id, which the launcher in DIR
# reads from that file:
#
#   <id>.shell   the frame: the hotkey and the panel
#   <id>.clipd   the clipboard watcher
#
# Running it again stops both and starts them on whatever DIR and FRAME hold
# now. Logs go to ~/Library/Logs/<name>. `scripts/uninstall` removes both.
#
# Written for the bash macOS ships, 3.2. The caller sets -euo pipefail.

swoop_launchd() {
  local dir="$1" frame="$2"
  local agents="$HOME/Library/LaunchAgents"
  local name id logs
  local domain fzf path label
  # The tool's name and id, from its tool file. The launcher reads it, so
  # nothing here spells either, and a tool with another name gets agents
  # and a log folder of its own.
  if ! name="$("$dir/bin/swoop" tool name)" || ! id="$("$dir/bin/swoop" tool id)"; then
    echo "launchd: $dir/bin/swoop tool did not say what this tool is" >&2; return 1
  fi
  logs="$HOME/Library/Logs/$name"
  domain="gui/$(id -u)"

  # fzf may be one of ours, in DIR/bin, rather than on the caller's PATH.
  fzf="$(PATH="$dir/bin:$PATH" command -v fzf)" || {
    echo "launchd: fzf is not on PATH or in $dir/bin" >&2; return 1; }

  # launchd gives an agent almost no environment. PATH must hold bin/ for the
  # tools the launcher runs and fzf's directory, which is not the same on every Mac.
  # When fzf is ours, its directory is DIR/bin, already first. After those,
  # the PATH of the shell running this install: an extension that runs a
  # tool of yours, like the AI command, finds it where your shell does.
  path="$dir/bin"
  [ "$(dirname "$fzf")" = "$dir/bin" ] || path="$path:$(dirname "$fzf")"
  path="$path:$PATH:/usr/local/bin:/usr/bin:/bin"
  mkdir -p "$agents" "$logs"

  # Stop, in this order. First unload both services, and wait until
  # launchd has forgotten them: a service that is still loaded has
  # KeepAlive, and a frame killed while its service is loaded is back
  # within a second, pre-warming a launcher that starts a watcher outside
  # launchd, which launchd's own watcher then finds and gives way to,
  # forever. Only once nothing can restart them, kill whatever is left of
  # the frame and the watcher, started by hand or left over, and wait
  # until they are gone too. Then the lock is free and the hotkey is
  # free, and the new agents start into a clean slate.
  local label _
  for label in "$id.shell" "$id.clipd"; do
    launchctl bootout "$domain/$label" >/dev/null 2>&1 || true
  done
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25; do
    launchctl print "$domain/$id.shell" >/dev/null 2>&1 || launchctl print "$domain/$id.clipd" >/dev/null 2>&1 || break
    sleep 0.2
  done
  pkill -x swoop-shell-mac 2>/dev/null || true
  pkill -f 'swoop-clipd run' 2>/dev/null || true
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25; do
    pgrep -x swoop-shell-mac >/dev/null 2>&1 || pgrep -f 'swoop-clipd run' >/dev/null 2>&1 || break
    sleep 0.2
  done

  _swoop_agent "$id.clipd" "$dir/bin/swoop-clipd" run
  # Until launchd's watcher holds the lock, the frame stays down: its
  # first launcher would otherwise start a watcher of its own in the second
  # launchd takes to spawn, and hold the lock from then on.
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25; do
    "$dir/bin/swoop-clipd" status >/dev/null 2>&1 && break
    sleep 0.2
  done
  _swoop_agent "$id.shell" "$frame"

  sleep 1
  for label in "$id.clipd" "$id.shell"; do
    if launchctl print "$domain/$label" 2>/dev/null | grep -q "state = running"; then
      echo "running $label"
    else
      echo "launchd: $label is not running; see $logs" >&2
      return 1
    fi
  done
}

# One agent per program. ProcessType Interactive keeps launchd from
# throttling the frame like a background job; the hotkey must answer at
# once. KeepAlive restarts either one if it ever dies. Reads agents, logs,
# domain, path, dir and id from swoop_launchd, which calls it.
_swoop_agent() { # label program args... (args after the program)
  local label="$1" program="$2"; shift 2
  local plist="$agents/$label.plist" args="" a
  for a in "$@"; do args="$args      <string>$a</string>
"; done
  cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$label</string>
  <key>ProgramArguments</key>
  <array>
      <string>$program</string>
$args  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key><string>$path</string>
    <key>SWOOP_LAUNCHER</key><string>$dir/bin/swoop</string>
${SWOOP_HOTKEY:+    <key>SWOOP_HOTKEY</key><string>$SWOOP_HOTKEY</string>
}  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>ProcessType</key><string>Interactive</string>
  <key>StandardOutPath</key><string>$logs/${label#"$id".}.log</string>
  <key>StandardErrorPath</key><string>$logs/${label#"$id".}.log</string>
</dict>
</plist>
EOF
  # The service was unloaded by swoop_launchd before this. bootstrap can
  # still be refused for a moment after an unload, so it is tried more
  # than once.
  for _ in 1 2 3 4 5; do
    if launchctl bootstrap "$domain" "$plist" 2>/dev/null; then
      echo "started $label"
      return 0
    fi
    sleep 0.5
  done
  echo "launchd: could not start $label; launchctl bootstrap keeps refusing" >&2
  return 1
}
