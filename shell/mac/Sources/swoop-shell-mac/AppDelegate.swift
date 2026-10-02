import AppKit
import GhosttyTerminal

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var launcher: LauncherController?
    private var hotKey: HotKey?
    private var toggleSignal: DispatchSourceSignal?
    private var endedSignal: DispatchSourceSignal?
    private var statusItem: StatusItem?
    private var configWatch: Timer?
    private var selfWatch: Timer?
    private var hotkeySpec = ""

    func applicationDidFinishLaunching(_: Notification) {
        // SWOOP_SHELL_DEBUG=1 logs every libghostty callback and every
        // show/hide to stderr: the way to see why a panel did or did not
        // close.
        if ProcessInfo.processInfo.environment["SWOOP_SHELL_DEBUG"] != nil {
            TerminalDebugLog.enable(.standard)
            TerminalDebugLog.sink = { FileHandle.standardError.write(Data(($0 + "\n").utf8)) }
        }
        guard let command = Launcher.path else {
            FileHandle.standardError.write(Data("swoop-shell-mac: cannot find the launcher. Set SWOOP_LAUNCHER or put bin/ on PATH.\n".utf8))
            NSApp.terminate(nil)
            return
        }
        // What this frame is: its name, title, id and hotkey, from the
        // tool file, through the launcher. Asked for here, before anything
        // reads a folder or shows a name. See Tool.
        let tool = Tool.current
        let launcher = LauncherController(command: command)
        self.launcher = launcher

        let spec = Config.hotkey()
        registerHotkey(spec)
        // The Settings pane writes a new hotkey to the settings file; the
        // frame takes it within a second, no restart.
        configWatch = Config.watch { [weak self] in
            guard let self else { return }
            let now = Config.hotkey()
            if now != self.hotkeySpec { self.registerHotkey(now) }
        }

        // A new frame at our own path (an upgrade, a rebuild): exit once
        // the panel is hidden, and the service starts the new one.
        selfWatch = SelfWatch.watch(busy: { [weak launcher] in launcher?.isShowing ?? false }) {
            FileHandle.standardError.write(Data("swoop-shell-mac: \(SelfWatch.path) changed; exiting so the service starts the new one\n".utf8))
            exit(0)
        }

        // The bird in the menu bar. SWOOP_NO_MENU_BAR=1 leaves it out.
        if ProcessInfo.processInfo.environment["SWOOP_NO_MENU_BAR"] == nil {
            statusItem = StatusItem(hotkey: spec) { [weak launcher] in launcher?.show() }
        }

        // SIGUSR1 toggles the panel: `kill -USR1 $(pgrep swoop-shell-mac)`.
        // A test can drive the frame without a keyboard, and a script can
        // open the launcher without knowing the hotkey.
        signal(SIGUSR1, SIG_IGN)
        let toggle = DispatchSource.makeSignalSource(signal: SIGUSR1, queue: .main)
        toggle.setEventHandler { [weak launcher] in launcher?.toggle() }
        toggle.resume()
        toggleSignal = toggle

        // SIGUSR2 is the launcher saying "I am about to exit": hide the panel and
        // start the next one. See LauncherController.replace.
        signal(SIGUSR2, SIG_IGN)
        let ended = DispatchSource.makeSignalSource(signal: SIGUSR2, queue: .main)
        ended.setEventHandler { [weak launcher] in launcher?.launcherEnded() }
        ended.resume()
        endedSignal = ended

        FileHandle.standardError.write(Data("swoop-shell-mac: ready as \(tool.title) (\(tool.id).shell, folder \(tool.name)); hotkey \(spec); running \(command)\n".utf8))
    }

    /// Register spec as the hotkey, in place of the one before. A key that
    /// cannot be taken leaves the frame without one, and the log says so:
    /// under launchd a quit is a restart loop, and SIGUSR1 still works.
    private func registerHotkey(_ spec: String) {
        hotKey = nil
        hotkeySpec = spec
        do {
            hotKey = try HotKey(spec) { [weak launcher] in
                DispatchQueue.main.async { launcher?.toggle() }
            }
            FileHandle.standardError.write(Data("swoop-shell-mac: hotkey \(spec)\n".utf8))
        } catch {
            FileHandle.standardError.write(Data("swoop-shell-mac: \(error); running without a hotkey\n".utf8))
        }
    }
}
