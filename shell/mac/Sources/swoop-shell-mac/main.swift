// swoop-shell-mac: the macOS frame. Run it and it waits in the background
// with no Dock icon. The hotkey shows a floating panel with the launcher in
// it; Esc, Enter, or a click elsewhere hides it; the next press starts a
// fresh one. SIGUSR1 toggles the panel too, for scripts and for tests.
//
// The frame is swoopkit's, and belongs to whichever tool its launcher
// runs as: the name, title, id and hotkey come from that tool's `tool`
// file, through `<launcher> tool`. See Tool.
//
//   SWOOP_LAUNCHER   path to bin/swoop; otherwise `swoop` is found on PATH
//   SWOOP_HOTKEY     "alt+space", "ctrl+space", ...; unset, the settings
//                    file's, then the tool's own
import AppKit

MainActor.assumeIsolated {
    let app = NSApplication.shared
    // Accessory: no Dock icon, no menu bar, nothing steals the front app.
    app.setActivationPolicy(.accessory)
    let delegate = AppDelegate()
    app.delegate = delegate
    app.run()
}
