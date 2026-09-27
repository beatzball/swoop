import Foundation

/// The launcher's own settings file, ~/.config/swoop/config: one
/// `key = value` per line, `#` a comment. The frame reads one key from it,
/// the hotkey, and watches the file so that a change made in the Settings
/// pane takes effect with no restart. The Go side (internal/settings)
/// writes it; this reads the same shape.
enum Config {
    static var path: String {
        let env = ProcessInfo.processInfo.environment
        let base = env["XDG_CONFIG_HOME"] ?? (env["HOME"] ?? "") + "/.config"
        return base + "/swoop/config"
    }

    /// The value of key in the file, or nil when the file or the key is
    /// not there.
    static func get(_ key: String) -> String? {
        guard let text = try? String(contentsOfFile: path, encoding: .utf8) else { return nil }
        for raw in text.split(separator: "\n", omittingEmptySubsequences: false) {
            let line = raw.trimmingCharacters(in: .whitespaces)
            if line.isEmpty || line.hasPrefix("#") { continue }
            guard let eq = line.firstIndex(of: "=") else { continue }
            let k = line[..<eq].trimmingCharacters(in: .whitespaces)
            if k == key {
                return line[line.index(after: eq)...].trimmingCharacters(in: .whitespaces)
            }
        }
        return nil
    }

    /// The hotkey to register: SWOOP_HOTKEY in the environment wins, for
    /// scripts and tests; then the file; then the default.
    static func hotkey() -> String {
        if let env = ProcessInfo.processInfo.environment["SWOOP_HOTKEY"], !env.isEmpty { return env }
        if let v = get("hotkey"), !v.isEmpty { return v }
        return "alt+shift+space"
    }

    /// Calls `changed` on the main queue whenever the file's modification
    /// time moves. A poll every second rather than a file watcher: the
    /// file may not exist yet, is rewritten in place, and a second is
    /// soon enough for a key someone just picked.
    static func watch(_ changed: @escaping () -> Void) -> Timer {
        var last = modified()
        let timer = Timer(timeInterval: 1.0, repeats: true) { _ in
            let now = modified()
            if now != last {
                last = now
                changed()
            }
        }
        RunLoop.main.add(timer, forMode: .common)
        return timer
    }

    private static func modified() -> Date? {
        (try? FileManager.default.attributesOfItem(atPath: path))?[.modificationDate] as? Date
    }
}
