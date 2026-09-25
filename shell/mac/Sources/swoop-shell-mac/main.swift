// swoop-shell-mac: the macOS frame. Run it and it waits in the background
// with no Dock icon. The hotkey shows a floating panel with swoop in it;
// Esc, Enter, or a click elsewhere hides it; the next press starts a fresh
// swoop. SIGUSR1 toggles the panel too, for scripts and for tests.
//
//   SWOOP_LAUNCHER   path to bin/swoop; otherwise `swoop` is found on PATH
//   SWOOP_HOTKEY     "alt+shift+space" (default), "alt+space", "ctrl+space", ...
import AppKit

MainActor.assumeIsolated {
    let app = NSApplication.shared
    // Accessory: no Dock icon, no menu bar, nothing steals the front app.
    app.setActivationPolicy(.accessory)
    let delegate = AppDelegate()
    app.delegate = delegate
    app.run()
}
