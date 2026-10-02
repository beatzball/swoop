import Foundation

/// The tool this frame belongs to: what its file `tool`, beside the
/// extensions folder, says. The name is the word in the folders, the
/// title is what the menu and a notification show, the id is the prefix
/// of the launchd labels and of the frame's signature, and the hotkey
/// opens the panel until the user picks a key of their own.
///
/// The frame spells none of them and has no reader of its own for the
/// file. It asks the launcher it runs, `<launcher> tool`, which prints
/// the four lines with everything the file leaves out filled in. One
/// frame binary so serves any tool built on the kit: point SWOOP_LAUNCHER
/// at that tool's launcher and the frame is that tool's.
struct Tool {
    let name: String
    let title: String
    let id: String
    let hotkey: String

    /// Read once, the first time it is asked for. A frame that cannot
    /// learn what it is has no folder to read its settings from and no
    /// name to show, so it says why and stops.
    static let current: Tool = {
        guard let launcher = Launcher.path else {
            fail("cannot find the launcher. Set SWOOP_LAUNCHER or put bin/ on PATH.")
        }
        let task = Process()
        task.executableURL = URL(fileURLWithPath: launcher)
        task.arguments = ["tool"]
        let out = Pipe()
        task.standardOutput = out
        // No terminal: a launcher too old to know the verb would open
        // fzf, which gives up at once with nothing to read from.
        task.standardInput = FileHandle.nullDevice
        do {
            try task.run()
        } catch {
            fail("\(launcher) tool: \(error)")
        }
        let data = out.fileHandleForReading.readDataToEndOfFile()
        task.waitUntilExit()
        var values: [String: String] = [:]
        for line in String(decoding: data, as: UTF8.self).split(separator: "\n") {
            let parts = line.split(separator: " ", maxSplits: 1)
            if parts.count == 2 { values[String(parts[0])] = String(parts[1]) }
        }
        guard task.terminationStatus == 0,
              let name = values["name"], let title = values["title"],
              let id = values["id"], let hotkey = values["hotkey"]
        else {
            fail("\(launcher) tool did not say what this tool is (status \(task.terminationStatus)); the launcher and the frame must be the same version")
        }
        return Tool(name: name, title: title, id: id, hotkey: hotkey)
    }()

    /// The tool's folder under the user's config home: the settings file,
    /// the frame's own file, the user's extensions.
    static var configDir: String {
        let env = ProcessInfo.processInfo.environment
        let base = env["XDG_CONFIG_HOME"] ?? (env["HOME"] ?? "") + "/.config"
        return base + "/" + current.name
    }

    private static func fail(_ message: String) -> Never {
        FileHandle.standardError.write(Data("swoop-shell-mac: \(message)\n".utf8))
        exit(1)
    }
}

/// The launcher script: SWOOP_LAUNCHER, or `swoop`, the kit's launcher, on
/// PATH. The frame never guesses a path of its own; it runs what it is
/// told to run.
enum Launcher {
    static let path: String? = {
        let env = ProcessInfo.processInfo.environment
        if let p = env["SWOOP_LAUNCHER"], FileManager.default.isExecutableFile(atPath: p) {
            return p
        }
        for dir in (env["PATH"] ?? "").split(separator: ":") {
            let p = String(dir) + "/swoop"
            if FileManager.default.isExecutableFile(atPath: p) {
                return p
            }
        }
        return nil
    }()
}
