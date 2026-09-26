import AppKit
import GhosttyTerminal

/// A borderless floating panel that can take the keyboard without making
/// this app the front app, which is what a launcher wants: it appears over
/// whatever you were doing and leaves it in place.
final class LauncherPanel: NSPanel {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }
}

/// Shows and hides the panel and owns the terminal surface inside it.
///
/// Hiding keeps the launcher as it is: the hotkey, or a click elsewhere,
/// puts the panel away with your text still in the bar, and the next press
/// brings the same launcher back. Only swoop ending, Esc at the root or
/// Enter's `become`, replaces it, and the replacement is started at once
/// while the panel is hidden, so a press never shows an empty terminal.
final class LauncherController: NSObject, NSWindowDelegate,
    TerminalSurfaceCloseDelegate, TerminalSurfaceCommandFinishedDelegate
{
    private let command: String
    private let panel: LauncherPanel
    private var terminal: TerminalView?
    private var settings: Settings
    private lazy var controller = TerminalController(configuration: configuration())
    private var keyMonitor: Any?

    /// The smallest panel that still shows a list and a preview.
    private static let minSize = NSSize(width: 480, height: 280)

    init(command: String) {
        self.command = command
        let loaded = Settings.load()
        settings = loaded
        panel = LauncherPanel(
            contentRect: NSRect(origin: .zero, size: loaded.size),
            // resizable: drag any edge. The size is kept in the settings
            // file when the drag ends; see windowDidEndLiveResize.
            styleMask: [.borderless, .nonactivatingPanel, .fullSizeContentView, .resizable],
            backing: .buffered,
            defer: false
        )
        super.init()
        panel.minSize = Self.minSize
        panel.level = .floating
        panel.isFloatingPanel = true
        panel.hidesOnDeactivate = false
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.isMovableByWindowBackground = false
        // Show on the space the user is on, including over a full-screen app.
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .transient]
        panel.delegate = self

        let content = NSView(frame: NSRect(origin: .zero, size: loaded.size))
        content.wantsLayer = true
        content.layer?.cornerRadius = 14
        content.layer?.masksToBounds = true
        content.layer?.backgroundColor = NSColor.black.withAlphaComponent(0.96).cgColor
        panel.contentView = content

        // cmd+plus, cmd+minus, cmd+0 change the font size and the change is
        // kept for next time. The frame takes these keys rather than
        // libghostty, because libghostty does not say what size it ended up
        // at, and a size that resets on every open is worse than none.
        keyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            guard let self, event.window === self.panel,
                  event.modifierFlags.intersection(.deviceIndependentFlagsMask).contains(.command)
            else { return event }
            switch event.charactersIgnoringModifiers {
            case "=", "+": self.changeFontSize(by: 1)
            case "-": self.changeFontSize(by: -1)
            case "0": self.changeFontSize(to: Settings.defaultFontSize)
            default: return event
            }
            return nil
        }

        prepare()
    }

    // MARK: configuration

    private func configuration() -> TerminalConfiguration {
        TerminalConfiguration { builder in
            builder.withFontSize(settings.fontSize)
            builder.withBackgroundOpacity(0.96)
            builder.withWindowPaddingX(12)
            builder.withWindowPaddingY(8)
            // The cursor is fzf's; a blinking block would fight with it.
            builder.withCursorStyleBlink(false)
            // The frame owns these keys; see the key monitor above.
            for key in ["super+equal", "super+plus", "super+minus", "super+zero"] {
                builder.withCustom("keybind", "\(key)=unbind")
            }
            // cmd+K is the action menu, as in Raycast. The terminal never
            // sees cmd keys, so the frame turns it into ctrl-k, which
            // bin/swoop binds.
            builder.withCustom("keybind", "super+k=text:\\x0b")
            // cmd+[ and cmd+] move the divider between list and preview:
            // alt+left and alt+right to fzf, sent as the escape sequences
            // a terminal sends for them.
            builder.withCustom("keybind", "super+bracket_left=text:\\x1b[1;3D")
            builder.withCustom("keybind", "super+bracket_right=text:\\x1b[1;3C")
        }
    }

    private func changeFontSize(by delta: Float) {
        changeFontSize(to: settings.fontSize + delta)
    }

    private func changeFontSize(to size: Float) {
        settings.fontSize = min(max(size, 8), 40)
        settings.save()
        _ = controller.setTerminalConfiguration(configuration())
    }

    // MARK: show and hide

    func toggle() {
        if panel.isVisible { hide() } else { show() }
    }

    /// Start a swoop in the hidden panel, ready to be shown.
    private func prepare() {
        guard terminal == nil, let content = panel.contentView else { return }
        let view = TerminalView(frame: content.bounds)
        view.autoresizingMask = [.width, .height]
        view.delegate = self
        view.configuration = TerminalSurfaceOptions(
            backend: .exec,
            // The frame's name and pid. swoop sends SIGUSR2 to the pid
            // just before it exits, on both of its exit paths, because
            // the library does not report the process ending (its close
            // callback is never wired up for the exec backend). The frame
            // then replaces the surface before the terminal can show
            // "Process exited".
            envVars: ["SWOOP_SHELL": "mac", "SWOOP_SHELL_PID": String(getpid())],
            command: command,
            waitAfterCommand: false
        )
        view.controller = controller
        content.addSubview(view)
        terminal = view
    }

    func show() {
        log("show")
        if terminal == nil { prepare() }
        centerOnActiveScreen()
        panel.makeKeyAndOrderFront(nil)
        if let terminal { panel.makeFirstResponder(terminal) }
    }

    /// Put the panel away and keep the launcher as it is.
    func hide() {
        log("hide")
        panel.orderOut(nil)
    }

    /// swoop said it is exiting (SIGUSR2). Replace it.
    func launcherEnded() {
        replace()
    }

    /// swoop ended: drop its surface and start the next one, hidden.
    private func replace() {
        log("replace")
        panel.orderOut(nil)
        terminal?.removeFromSuperview()
        terminal = nil
        prepare()
    }

    private func log(_ message: String) {
        guard ProcessInfo.processInfo.environment["SWOOP_SHELL_DEBUG"] != nil else { return }
        FileHandle.standardError.write(Data("swoop-shell-mac: \(message)\n".utf8))
    }

    private func centerOnActiveScreen() {
        // The screen with the mouse: where the user is looking.
        let mouse = NSEvent.mouseLocation
        let screen = NSScreen.screens.first { $0.frame.contains(mouse) } ?? NSScreen.main
        guard let frame = screen?.visibleFrame else { return }
        let size = panel.frame.size
        let origin = NSPoint(
            x: frame.midX - size.width / 2,
            // A little above centre, like Spotlight.
            y: frame.midY - size.height / 2 + frame.height * 0.08
        )
        panel.setFrame(NSRect(origin: origin, size: size), display: true)
    }

    // MARK: the surface says it is done

    func terminalDidFinishCommand(exitCode: Int?, durationNanos _: UInt64) {
        // Shell integration's "a command finished", not the process ending;
        // swoop is not a shell, so this is not expected. Logged, not acted on.
        log("command finished, exit \(exitCode.map(String.init) ?? "nil")")
    }

    func terminalDidClose(processAlive: Bool) {
        // swoop exited: Esc at the root, or Enter's become finished. The
        // core asks for the surface to close; replace it.
        log("close, processAlive \(processAlive)")
        DispatchQueue.main.async { self.replace() }
    }

    // MARK: the panel was resized

    func windowDidEndLiveResize(_: Notification) {
        // The terminal view follows the panel on its own (autoresizing);
        // what is left is to remember the size for next time.
        settings.width = Double(panel.frame.width)
        settings.height = Double(panel.frame.height)
        settings.save()
        log("resized to \(Int(settings.width)) by \(Int(settings.height))")
    }

    // MARK: the panel lost the keyboard

    func windowDidResignKey(_: Notification) {
        // A click into another app: the launcher goes away, like Raycast.
        if panel.isVisible { hide() }
    }
}

/// What the frame remembers between runs: the font size and the panel's
/// size. A small JSON file under the config directory, next to the
/// extensions. Every field has a default, so a file from before a field
/// existed still loads, and a hand-edited one can leave fields out.
struct Settings: Codable {
    static let defaultFontSize: Float = 16
    static let defaultWidth: Double = 880
    static let defaultHeight: Double = 520
    var fontSize: Float = Settings.defaultFontSize
    var width: Double = Settings.defaultWidth
    var height: Double = Settings.defaultHeight

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        fontSize = try c.decodeIfPresent(Float.self, forKey: .fontSize) ?? Settings.defaultFontSize
        width = try c.decodeIfPresent(Double.self, forKey: .width) ?? Settings.defaultWidth
        height = try c.decodeIfPresent(Double.self, forKey: .height) ?? Settings.defaultHeight
    }

    /// The panel's size from the file, never smaller than the frame's minimum.
    var size: NSSize {
        NSSize(width: max(width, 480), height: max(height, 280))
    }

    static var path: String {
        let env = ProcessInfo.processInfo.environment
        let base = env["XDG_CONFIG_HOME"] ?? (env["HOME"] ?? "") + "/.config"
        return base + "/swoop/shell-mac.json"
    }

    static func load() -> Settings {
        guard let data = FileManager.default.contents(atPath: path),
              let s = try? JSONDecoder().decode(Settings.self, from: data)
        else { return Settings() }
        return s
    }

    func save() {
        let dir = (Settings.path as NSString).deletingLastPathComponent
        try? FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        if let data = try? JSONEncoder().encode(self) {
            FileManager.default.createFile(atPath: Settings.path, contents: data)
        }
    }
}
