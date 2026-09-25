# shellcheck shell=bash
# launchd.sh: the one place that writes swoop's launchd agents. Sourced, not
# run, by both ways in: `scripts/install` (a checkout, through `make install`)
# and `scripts/get` (a downloaded release). Two copies of a plist would drift;
# this way a fix to one is a fix to both. It ships in the release tarball, so
# `scripts/get` needs nothing from a checkout.
#
#   swoop_launchd DIR FRAME
#
# DIR holds bin/swoop and bin/swoop-clipd: a checkout, or an install such as
# ~/.local/share/swoop/current. FRAME is the frame's executable, with its
# resource bundle beside it. Two agents run at login and keep running:
#
#   dev.swoop.shell   the frame: the hotkey and the panel
#   dev.swoop.clipd   the clipboard watcher
#
# Running it again stops both and starts them on whatever DIR and FRAME hold
# now. Logs go to ~/Library/Logs/swoop. `scripts/uninstall` removes both.
#
# Written for the bash macOS ships, 3.2. The caller sets -euo pipefail.

swoop_launchd() {
  local dir="$1" frame="$2"
  local agents="$HOME/Library/LaunchAgents"
  local logs="$HOME/Library/Logs/swoop"
  local domain fzf path label
  domain="gui/$(id -u)"

  # fzf may be one of ours, in DIR/bin, rather than on the caller's PATH.
  fzf="$(PATH="$dir/bin:$PATH" command -v fzf)" || {
    echo "launchd: fzf is not on PATH or in $dir/bin" >&2; return 1; }

  # launchd gives an agent almost no environment. PATH must hold bin/ for the
  # tools swoop runs and fzf's directory, which is not the same on every Mac.
  # When fzf is ours, its directory is DIR/bin, already first.
  path="$dir/bin"
  [ "$(dirname "$fzf")" = "$dir/bin" ] || path="$path:$(dirname "$fzf")"
  path="$path:/usr/local/bin:/usr/bin:/bin"
  mkdir -p "$agents" "$logs"

  # A frame or a watcher started by hand would keep the hotkey or the lock,
  # and launchd would restart its own copy in a loop. Stop them first.
  pkill -x swoop-shell-mac 2>/dev/null || true
  pkill -f 'swoop-clipd run' 2>/dev/null || true
  sleep 0.3

  _swoop_agent dev.swoop.clipd "$dir/bin/swoop-clipd" run
  _swoop_agent dev.swoop.shell "$frame"

  sleep 1
  for label in dev.swoop.clipd dev.swoop.shell; do
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
# domain, path and dir from swoop_launchd, which calls it.
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
  <key>StandardOutPath</key><string>$logs/${label#dev.swoop.}.log</string>
  <key>StandardErrorPath</key><string>$logs/${label#dev.swoop.}.log</string>
</dict>
</plist>
EOF
  # Stop the old one, if any, then start the new. bootout returns non-zero
  # when there is nothing to stop; that is the normal first run.
  launchctl bootout "$domain/$label" >/dev/null 2>&1 || true
  launchctl bootstrap "$domain" "$plist"
  echo "started $label"
}
