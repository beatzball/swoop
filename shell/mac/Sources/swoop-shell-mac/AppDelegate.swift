import AppKit
import GhosttyTerminal

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var launcher: LauncherController?
    private var hotKey: HotKey?
    private var toggleSignal: DispatchSourceSignal?
    private var endedSignal: DispatchSourceSignal?

    func applicationDidFinishLaunching(_: Notification) {
        // SWOOP_SHELL_DEBUG=1 logs every libghostty callback and every
        // show/hide to stderr: the way to see why a panel did or did not
        // close.
        if ProcessInfo.processInfo.environment["SWOOP_SHELL_DEBUG"] != nil {
            TerminalDebugLog.enable(.standard)
            TerminalDebugLog.sink = { FileHandle.standardError.write(Data(($0 + "\n").utf8)) }
        }
        guard let command = findLauncher() else {
            FileHandle.standardError.write(Data("swoop-shell-mac: cannot find swoop. Set SWOOP_LAUNCHER or put bin/ on PATH.\n".utf8))
            NSApp.terminate(nil)
            return
        }
        let launcher = LauncherController(command: command)
        self.launcher = launcher

        let spec = ProcessInfo.processInfo.environment["SWOOP_HOTKEY"] ?? "alt+shift+space"
        do {
            hotKey = try HotKey(spec) { [weak launcher] in launcher?.toggle() }
        } catch {
            FileHandle.standardError.write(Data("swoop-shell-mac: \(error)\n".utf8))
            NSApp.terminate(nil)
            return
        }

        // SIGUSR1 toggles the panel: `kill -USR1 $(pgrep swoop-shell-mac)`.
        // A test can drive the frame without a keyboard, and a script can
        // open the launcher without knowing the hotkey.
        signal(SIGUSR1, SIG_IGN)
        let toggle = DispatchSource.makeSignalSource(signal: SIGUSR1, queue: .main)
        toggle.setEventHandler { [weak launcher] in launcher?.toggle() }
        toggle.resume()
        toggleSignal = toggle

        // SIGUSR2 is swoop saying "I am about to exit": hide the panel and
        // start the next one. See LauncherController.replace.
        signal(SIGUSR2, SIG_IGN)
        let ended = DispatchSource.makeSignalSource(signal: SIGUSR2, queue: .main)
        ended.setEventHandler { [weak launcher] in launcher?.launcherEnded() }
        ended.resume()
        endedSignal = ended

        FileHandle.standardError.write(Data("swoop-shell-mac: ready; hotkey \(spec); running \(command)\n".utf8))
    }

    /// The launcher script: SWOOP_LAUNCHER, or `swoop` on PATH. The frame
    /// never guesses a path of its own; it runs what it is told to run.
    private func findLauncher() -> String? {
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
    }
}
