# shellcheck shell=bash
# launchd.sh: the one place that writes the tool's launchd agents. Sourced, not
# run, by every way in: `scripts/install` (a checkout, through `make install`),
# `scripts/get` (a downloaded release), and the launcher's `<name> start` for
# a tool made by `<name> new`. Two copies of a plist would drift; this way a
# fix to one is a fix to all. It ships in the release tarball, so
# `scripts/get` needs nothing from a checkout.
#
#   swoop_launchd DIR FRAME [LAUNCHER TOOL]
#
# DIR holds bin/swoop, bin/swoop-clipd and the file `tool`: a checkout, or
# an install such as ~/.local/share/<name>/current. FRAME is the frame's
# executable, with its resource bundle beside it. Two agents run at login
# and keep running, labelled with the tool's id, which the launcher
# reads from that file:
#
#   <id>.shell   the frame: the hotkey and the panel
#   <id>.clipd   the clipboard watcher
#
# LAUNCHER and TOOL are for a tool that is not DIR's own: a folder with a
# tool file and a command of its own, which runs the kit's programs in DIR.
# LAUNCHER is that command, the one its frame runs, and TOOL is its tool
# file, which the agents get as SWOOP_TOOL: the watcher has no command to
# tell it which tool it is. Without them the launcher is DIR/bin/swoop and
# the file is the one beside it.
#
# Running it again stops both and starts them on whatever DIR and FRAME hold
# now. Logs go to ~/Library/Logs/<name>. `scripts/uninstall` removes both.
#
# Two tools on one Mac run the same two programs. So nothing here stops a
# process by the name of its program: that would stop every tool's. A
# service is the tool's by its label, <id>.shell. A frame started by hand
# is the tool's by the launcher it runs (swoop_frames). A watcher is the
# tool's by the lock it holds, in the tool's own folder (swoop-clipd pid).
#
# Written for the bash macOS ships, 3.2. The caller sets -euo pipefail.

# swoop_frames LAUNCHER FRAME prints the process id of every frame that
# runs LAUNCHER, one a line: a frame under launchd, and one started by
# hand with SWOOP_LAUNCHER set. The launcher is what makes a frame one
# tool's: its path is in the frame's environment, which ps -E shows for
# this user's processes. Everything a frame starts has that variable too,
# and so may a shell someone exported it in, so a process counts only
# when its program has the file name FRAME has.
swoop_frames() {
  ps -x -Eww -o pid=,command= 2>/dev/null | awk -v want="SWOOP_LAUNCHER=$1" -v frame="${2##*/}" '
    {
      n = split($2, part, "/")
      if (part[n] != frame) next
      for (i = 3; i <= NF; i++) if ($i == want) { print $1; next }
    }'
}

# swoop_own_frame SRC NAME ID makes the frame of a tool that runs the
# kit's programs from somewhere else: a copy of the kit's frame SRC, with
# the resource bundles beside it, in ~/.local/share/NAME/frame, signed as
# ID.shell. It prints the copy's path.
#
# Why a copy: macOS keeps an Accessibility grant for a program file and
# its signature. Two tools running the one file would be one entry in
# System Settings and one grant. Its own file under its own identifier
# makes each tool's frame its own entry.
#
# Signed with <NAME>-dev when this Mac has that certificate, as
# scripts/sign-frame does, so the grant survives a new version of the kit;
# otherwise ad-hoc, and the grant holds until the kit's frame changes.
swoop_own_frame() {
  local src="$1" name="$2" id="$3"
  local to="$HOME/.local/share/$name/frame" b cert="$name-dev"
  rm -rf "$to.new"
  if ! mkdir -p "$to.new" || ! cp "$src" "$to.new/"; then
    echo "launchd: could not copy $src to $to.new" >&2; return 1
  fi
  # Every bundle: the frame looks for its resources beside itself, and
  # stops at launch without them.
  for b in "${src%/*}"/*.bundle; do
    [ -e "$b" ] || continue
    cp -R "$b" "$to.new/" || { echo "launchd: could not copy $b" >&2; return 1; }
  done
  # grep to /dev/null, not grep -q: -q stops reading at the first match,
  # security dies writing the rest, and pipefail calls that a failure.
  if ! security find-identity -v -p codesigning 2>/dev/null | grep "\"$cert\"" >/dev/null; then
    cert=-
  fi
  codesign -f -s "$cert" -i "$id.shell" "$to.new/${src##*/}" >&2 || {
    echo "launchd: could not sign $to.new/${src##*/} as $id.shell" >&2; return 1; }
  # Swapped whole, by name: a frame still running from the old copy keeps
  # the file it started from, and never runs one half written.
  rm -rf "$to"
  mv "$to.new" "$to" || return 1
  echo "$to/${src##*/}"
}

swoop_launchd() {
  local dir="$1" frame="$2" launcher="${3:-$1/bin/swoop}" tool="${4:-}"
  local agents="$HOME/Library/LaunchAgents"
  local name id logs
  local domain fzf path label
  # The tool's name and id, from its tool file. The launcher reads it, so
  # nothing here spells either, and a tool with another name gets agents
  # and a log folder of its own.
  if ! name="$("$launcher" tool name)" || ! id="$("$launcher" tool id)"; then
    echo "launchd: $launcher tool did not say what this tool is" >&2; return 1
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
  # this tool's frame and watcher, started by hand or left over, and wait
  # until they are gone too. Then the lock is free and the hotkey is
  # free, and the new agents start into a clean slate.
  #
  # This tool's, and no other's: another tool's frame and watcher are the
  # same two programs, and they keep running. See the top of this file.
  local label pid _
  for label in "$id.shell" "$id.clipd"; do
    launchctl bootout "$domain/$label" >/dev/null 2>&1 || true
  done
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25; do
    launchctl print "$domain/$id.shell" >/dev/null 2>&1 || launchctl print "$domain/$id.clipd" >/dev/null 2>&1 || break
    sleep 0.2
  done
  for pid in $(swoop_frames "$launcher" "$frame"); do
    kill "$pid" 2>/dev/null || true
  done
  if pid="$(_swoop_clipd pid 2>/dev/null)" && [ -n "$pid" ]; then
    kill "$pid" 2>/dev/null || true
  elif _swoop_clipd status >/dev/null 2>&1; then
    # The lock is held and nobody signed it: a watcher from a version
    # before the lock file named its holder. It cannot be told from
    # another tool's, so this once every watcher is stopped, the way it
    # was done then. Another tool's comes back by itself: launchd starts
    # it again at once, and a launcher starts one the next time it opens.
    pkill -f 'swoop-clipd run' 2>/dev/null || true
  fi
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25; do
    [ -n "$(swoop_frames "$launcher" "$frame")" ] || _swoop_clipd status >/dev/null 2>&1 || break
    sleep 0.2
  done

  _swoop_agent "$id.clipd" "$dir/bin/swoop-clipd" run
  # Until launchd's watcher holds the lock, the frame stays down: its
  # first launcher would otherwise start a watcher of its own in the second
  # launchd takes to spawn, and hold the lock from then on.
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25; do
    _swoop_clipd status >/dev/null 2>&1 && break
    sleep 0.2
  done
  _swoop_agent "$id.shell" "$frame"

  # launchd spawns each agent in its own time, and the frame asks the
  # launcher for its name before it is up: wait on the state, up to ten
  # seconds each, never on a fixed pause.
  # grep to /dev/null, not grep -q: -q stops reading at its match, and
  # under pipefail a launchctl still writing would count as a failure.
  for label in "$id.clipd" "$id.shell"; do
    for _ in $(seq 1 50); do
      launchctl print "$domain/$label" 2>/dev/null | grep "state = running" >/dev/null && break
      sleep 0.2
    done
    if launchctl print "$domain/$label" 2>/dev/null | grep "state = running" >/dev/null; then
      echo "running $label"
    else
      echo "launchd: $label is not running; see $logs" >&2
      return 1
    fi
  done
}

# The watcher's own program, asked about the tool being installed: the
# tool file it is given, or the one beside it. Reads dir and tool from
# swoop_launchd, which calls it.
_swoop_clipd() {
  if [ -n "$tool" ]; then
    SWOOP_TOOL="$tool" "$dir/bin/swoop-clipd" "$@"
  else
    "$dir/bin/swoop-clipd" "$@"
  fi
}

# One agent per program. ProcessType Interactive keeps launchd from
# throttling the frame like a background job; the hotkey must answer at
# once. KeepAlive restarts either one if it ever dies. Reads agents, logs,
# domain, path, launcher, tool and id from swoop_launchd, which calls it.
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
    <key>SWOOP_LAUNCHER</key><string>$launcher</string>
${tool:+    <key>SWOOP_TOOL</key><string>$tool</string>
}${SWOOP_HOTKEY:+    <key>SWOOP_HOTKEY</key><string>$SWOOP_HOTKEY</string>
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
