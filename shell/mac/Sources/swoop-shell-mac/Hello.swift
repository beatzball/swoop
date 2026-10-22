import Foundation

/// The one note a fresh install gets: "<title> is running. Press
/// <hotkey>." A frame started by `brew services start` or by the
/// installer puts nothing on screen, so the person who just ran the
/// right command has no way to tell it worked (#268). Said once per
/// machine, not once per start: a marker in the tool's state folder
/// says it has been said, and a restart or an upgrade is not a first
/// time.
enum Hello {
    /// The marker. In the state folder, not the config folder: it is
    /// something the frame remembers, not a setting of the user's.
    static var path: String { Tool.stateDir + "/hello" }

    /// Posts the note if this is the first start. The marker is written
    /// before the note goes out, so a frame that stops right after does
    /// not say it again at every start launchd gives it.
    static func once(hotkey: String) {
        let fm = FileManager.default
        if fm.fileExists(atPath: path) { return }
        try? fm.createDirectory(atPath: (path as NSString).deletingLastPathComponent, withIntermediateDirectories: true)
        // No folder to remember it in: say nothing, rather than say it
        // at every start.
        guard fm.createFile(atPath: path, contents: Data()) else { return }
        Paster.notify("\(Tool.current.title) is running. Press \(hotkey).")
        FileHandle.standardError.write(Data("swoop-shell-mac: first start on this machine; said hello\n".utf8))
    }
}
