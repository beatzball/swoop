import Foundation

/// The frame's own executable, watched so that an upgrade takes effect
/// with no restart by hand. `brew upgrade`, a `make install` that rebuilt
/// the frame, or the curl installer run again all leave a new file at the
/// path the frame was started from; the frame sees it, waits until the
/// panel is hidden, and exits 0. Its service (launchd's KeepAlive, or
/// Homebrew's, which is the same) starts the new one.
enum SelfWatch {
    /// What says "a different file": modification time, size, and inode.
    /// An install that swaps the file by rename keeps the time it was
    /// built, so the inode catches what the time would miss.
    private struct Stamp: Equatable {
        let modified: Date?
        let size: UInt64?
        let inode: UInt64?
    }

    /// The path the frame was started from, as launchd or a shell gave
    /// it, not with its symlinks resolved: Homebrew's service runs
    /// opt/swoop/..., a link an upgrade moves to the new version, and the
    /// curl installer's runs current/..., the same. Relative only when
    /// run by hand, so made whole against the directory it started in.
    static var path: String {
        let first = CommandLine.arguments.first ?? ""
        if first.hasPrefix("/") { return first }
        return FileManager.default.currentDirectoryPath + "/" + first
    }

    /// Calls `restart` on the main queue once the executable has changed
    /// and it is safe to go: the new file is there and has held still for
    /// a poll, so a copy still in progress is never what launchd starts,
    /// and `busy` (the panel is showing) has been false for two polls in
    /// a row, so a paste the launcher just asked for has time to land.
    /// The same one-second poll as Config.watch, for the same reasons.
    static func watch(busy: @escaping () -> Bool, restart: @escaping () -> Void) -> Timer {
        let first = stamp()
        var last = first
        var idle = 0
        let timer = Timer(timeInterval: 1.0, repeats: true) { _ in
            let now = stamp()
            defer { last = now }
            idle = busy() ? 0 : idle + 1
            guard now != first, now.modified != nil, now == last, idle >= 2 else { return }
            restart()
        }
        RunLoop.main.add(timer, forMode: .common)
        return timer
    }

    private static func stamp() -> Stamp {
        let attrs = try? FileManager.default.attributesOfItem(atPath: path)
        return Stamp(
            modified: attrs?[.modificationDate] as? Date,
            size: (attrs?[.size] as? NSNumber)?.uint64Value,
            inode: (attrs?[.systemFileNumber] as? NSNumber)?.uint64Value
        )
    }
}
