import AppKit

/// The bird in the menu bar: the one visible sign that swoop is running,
/// as Raycast and the other launchers have. Its menu opens the launcher,
/// opens the settings folder, and quits. An accessory app has no Dock
/// icon, so without this there is nothing to click.
final class StatusItem {
    private let item: NSStatusItem
    private let open: () -> Void
    private let hotkey: String

    init(hotkey: String, open: @escaping () -> Void) {
        self.open = open
        self.hotkey = hotkey
        item = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        if let button = item.button {
            // A template image takes the menu bar's own colour, light or
            // dark. The bird is an SF Symbol; a system without it (older
            // macOS) gets the name as text.
            if let image = NSImage(systemSymbolName: "bird", accessibilityDescription: "swoop") {
                image.isTemplate = true
                button.image = image
            } else {
                button.title = "swoop"
            }
            button.toolTip = "swoop  \(hotkey)"
        }
        let menu = NSMenu()
        let openItem = NSMenuItem(title: "Open swoop", action: #selector(openLauncher), keyEquivalent: "")
        openItem.target = self
        menu.addItem(openItem)
        let hint = NSMenuItem(title: hotkey, action: nil, keyEquivalent: "")
        hint.isEnabled = false
        menu.addItem(hint)
        menu.addItem(.separator())
        let settings = NSMenuItem(title: "Settings…", action: #selector(openSettings), keyEquivalent: "")
        settings.target = self
        menu.addItem(settings)
        menu.addItem(.separator())
        let quit = NSMenuItem(title: "Quit swoop", action: #selector(quit), keyEquivalent: "")
        quit.target = self
        menu.addItem(quit)
        item.menu = menu
    }

    @objc private func openLauncher() { open() }

    /// The config folder in Finder: the settings files, the clipboard
    /// ignore list, and a place for extensions of your own.
    @objc private func openSettings() {
        let env = ProcessInfo.processInfo.environment
        let base = env["XDG_CONFIG_HOME"] ?? (env["HOME"] ?? "") + "/.config"
        let dir = base + "/swoop"
        try? FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        NSWorkspace.shared.open(URL(fileURLWithPath: dir))
    }

    /// Under launchd, a quit is followed by a restart: KeepAlive. The
    /// menu says so, since a person who meant "stop it" needs the
    /// uninstall instead.
    @objc private func quit() {
        NSApp.terminate(nil)
    }
}
