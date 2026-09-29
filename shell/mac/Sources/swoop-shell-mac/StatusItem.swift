import AppKit

/// The bird in the menu bar: the one visible sign that swoop is running,
/// as the launchers you know have. Its menu opens the launcher,
/// opens the settings folder, restarts, and quits. An accessory app has no Dock
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
            // dark. The owl is a silhouette cut from the logo, 18 px with a
            // 36 px @2x, in the resource bundle; without the bundle the SF
            // Symbol bird stands in, and without that the name as text.
            if let image = StatusItem.owl() {
                button.image = image
            } else if let image = NSImage(systemSymbolName: "bird", accessibilityDescription: "swoop") {
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
        let restart = NSMenuItem(title: "Restart swoop", action: #selector(restart), keyEquivalent: "")
        restart.target = self
        menu.addItem(restart)
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

    /// Exit, and let the service start a new frame: launchd's KeepAlive,
    /// or Homebrew's service, which is the same thing. A frame run by
    /// hand has no service, and stays down; the log says which.
    @objc private func restart() {
        let label = Self.serviceLabel()
        log(label.map { "restart from the menu; \($0) starts a new frame" }
            ?? "restart from the menu; no service to start a new frame, so this is a quit")
        exit(0)
    }

    /// Stop, and stay stopped until the next login or `swoop start`. An
    /// exit alone is not enough under a service: KeepAlive would start
    /// the frame again within a second. So the frame unloads its own
    /// service with `launchctl bootout`, which ends this process too.
    /// The plist stays, so the next login starts it as before.
    @objc private func quit() {
        guard let label = Self.serviceLabel() else {
            log("quit from the menu; not under a service, exiting")
            exit(0)
        }
        log("quit from the menu; launchctl bootout of \(label)")
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/launchctl")
        task.arguments = ["bootout", "gui/\(getuid())/\(label)"]
        do {
            try task.run()
            task.waitUntilExit()
        } catch {
            log("quit: launchctl: \(error)")
        }
        // Only reached when bootout did not end us: it refused, or the
        // label was not loaded after all.
        log("quit: launchctl bootout of \(label) did not stop the frame (status \(task.terminationStatus)); exiting, and the service may start it again")
        exit(0)
    }

    /// The launchd label this frame runs under, when a plist for it is in
    /// ~/Library/LaunchAgents: dev.swoop.shell from the installers,
    /// homebrew.mxcl.swoop from `brew services`. launchd names the job in
    /// XPC_SERVICE_NAME; a frame run from a terminal has none, or "0".
    private static func serviceLabel() -> String? {
        let env = ProcessInfo.processInfo.environment
        guard let label = env["XPC_SERVICE_NAME"], !label.isEmpty, label != "0" else { return nil }
        let plist = (env["HOME"] ?? "") + "/Library/LaunchAgents/\(label).plist"
        return FileManager.default.fileExists(atPath: plist) ? label : nil
    }

    private func log(_ message: String) {
        FileHandle.standardError.write(Data("swoop-shell-mac: \(message)\n".utf8))
    }

    /// The owl from the resource bundle, as a template image sized for
    /// the bar: 18 points, with the @2x file for Retina. nil when the
    /// bundle is not beside the executable.
    static func owl() -> NSImage? {
        guard let url = Bundle.module.url(forResource: "menubar", withExtension: "png", subdirectory: "Resources"),
              let image = NSImage(contentsOf: url) else { return nil }
        if let url2x = Bundle.module.url(forResource: "menubar@2x", withExtension: "png", subdirectory: "Resources"),
           let rep = NSImageRep.imageReps(withContentsOf: url2x)?.first {
            rep.size = NSSize(width: 18, height: 18)
            image.addRepresentation(rep)
        }
        image.size = NSSize(width: 18, height: 18)
        image.isTemplate = true
        image.accessibilityDescription = "swoop"
        return image
    }
}
