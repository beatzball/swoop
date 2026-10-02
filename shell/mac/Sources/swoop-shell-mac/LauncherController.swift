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
/// brings the same launcher back. Only the launcher ending, Esc at the root or
/// Enter's `become`, replaces it, and the replacement is started at once
/// while the panel is hidden, so a press never shows an empty terminal.
final class LauncherController: NSObject, NSWindowDelegate,
    TerminalSurfaceCloseDelegate, TerminalSurfaceCommandFinishedDelegate
{
    private let command: String
    private let panel: LauncherPanel
    private var terminal: TerminalView?
    private var settings: Settings
    private var edgeHint: EdgeHintView?
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
        // The edge hint sits above everything else in the panel and takes
        // no clicks; see EdgeHintView. Mouse movement must reach it.
        let hintView = EdgeHintView(frame: content.bounds)
        hintView.autoresizingMask = [.width, .height]
        content.addSubview(hintView, positioned: .above, relativeTo: nil)
        edgeHint = hintView
        panel.acceptsMouseMovedEvents = true

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
            // cmd+K is the action menu, as in the launchers you know. The terminal never
            // sees cmd keys, so the frame turns it into ctrl-k, which
            // bin/swoop binds.
            builder.withCustom("keybind", "super+k=text:\\x0b")
            // cmd+[ and cmd+] move the divider between list and preview:
            // alt+left and alt+right to fzf, sent as the escape sequences
            // a terminal sends for them.
            builder.withCustom("keybind", "super+bracket_left=text:\\x1b[1;3D")
            builder.withCustom("keybind", "super+bracket_right=text:\\x1b[1;3C")
            // cmd+, opens Settings, the Mac convention: alt+, to fzf.
            builder.withCustom("keybind", "super+comma=text:\\x1b,")
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

    /// Whether the panel is on screen. The frame never restarts itself
    /// while it is: see SelfWatch.
    var isShowing: Bool { panel.isVisible }

    /// Start a launcher in the hidden panel, ready to be shown.
    private func prepare() {
        guard terminal == nil, let content = panel.contentView else { return }
        Paster.discard()
        let view = TerminalView(frame: content.bounds)
        view.autoresizingMask = [.width, .height]
        view.delegate = self
        view.configuration = TerminalSurfaceOptions(
            backend: .exec,
            // The frame's name and pid. The launcher sends SIGUSR2 to the pid
            // just before it exits, on both of its exit paths, because
            // the library does not report the process ending (its close
            // callback is never wired up for the exec backend). The frame
            // then replaces the surface before the terminal can show
            // "Process exited".
            //
            // SWOOP_PASTE is where the launcher leaves text to paste; the
            // frame reads it on that same signal. See Paster.
            envVars: ["SWOOP_SHELL": "mac", "SWOOP_SHELL_PID": String(getpid()), "SWOOP_PASTE": Paster.requestPath],
            command: command,
            waitAfterCommand: false
        )
        view.controller = controller
        // Below the edge hint, so the hint stays on top of the terminal.
        content.addSubview(view, positioned: .below, relativeTo: edgeHint)
        terminal = view
    }

    func show() {
        let start = DispatchTime.now()
        log("show")
        if terminal == nil { prepare() }
        centerOnActiveScreen()
        panel.makeKeyAndOrderFront(nil)
        if let terminal { panel.makeFirstResponder(terminal) }
        // The frame's own cost of a show, for scripts/bench, which reads
        // this line: what the key press waits on before the panel is the
        // system's to draw.
        log("shown in \(Self.milliseconds(since: start)) ms")
    }

    /// The time since start as milliseconds with two decimals, for the log.
    private static func milliseconds(since start: DispatchTime) -> String {
        let nanos = DispatchTime.now().uptimeNanoseconds - start.uptimeNanoseconds
        return String(format: "%.2f", Double(nanos) / 1_000_000)
    }

    /// Press key in the launcher, by fzf's name for the key, and show the
    /// panel first when it is hidden: how the menu's Settings… opens the
    /// Settings pane. The launcher in a hidden panel is already running,
    /// so the key lands on it as it would on an open one. A key with no
    /// bytes here is not pressed; false says so.
    @discardableResult
    func press(_ key: String) -> Bool {
        guard let text = Self.text(for: key) else {
            log("press \(key): not a key the frame can send")
            return false
        }
        if !panel.isVisible { show() }
        // A text binding, the same way cmd+, is sent: straight to the
        // launcher. The view's paste would wrap the bytes as pasted text,
        // and fzf would put them in the bar.
        let sent = terminal?.performBindingAction("text:" + text) ?? false
        log("press \(key), sent \(sent)")
        return sent
    }

    /// What a terminal sends for a key an extension may claim, written as
    /// a `text:` binding takes it: tab, shift-tab, f1 to f12, and alt with
    /// one lower-case letter, digit, comma, full stop or slash. The same
    /// set as ext.ValidKey, and nil for anything else.
    static func text(for key: String) -> String? {
        switch key {
        case "tab": return "\\x09"
        case "shift-tab": return "\\x1b[Z"
        default: break
        }
        let function = ["OP", "OQ", "OR", "OS", "[15~", "[17~", "[18~", "[19~", "[20~", "[21~", "[23~", "[24~"]
        if key.hasPrefix("f"), let n = Int(key.dropFirst()), function.indices.contains(n - 1) {
            return "\\x1b" + function[n - 1]
        }
        if key.hasPrefix("alt-"), key.utf8.count == 5, let c = key.unicodeScalars.last,
           ("a"..."z").contains(c) || ("0"..."9").contains(c) || ",./".unicodeScalars.contains(c)
        {
            // alt is Esc and then the key.
            return "\\x1b" + String(c)
        }
        return nil
    }

    /// Put the panel away and keep the launcher as it is.
    func hide() {
        let start = DispatchTime.now()
        log("hide")
        panel.orderOut(nil)
        // As in show: scripts/bench reads this line.
        log("hidden in \(Self.milliseconds(since: start)) ms")
    }

    /// The launcher said it is exiting (SIGUSR2). Replace it, and once the panel
    /// is off screen, paste what it asked for, if it asked.
    func launcherEnded() {
        // Taken first: replace starts the next launcher, and that clears
        // any request left over.
        let request = Paster.take()
        replace()
        if let request { Paster.paste(request) }
    }

    /// The launcher ended: drop its surface and start the next one, hidden.
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
        // The launcher is not a shell, so this is not expected. Logged, not acted on.
        log("command finished, exit \(exitCode.map(String.init) ?? "nil")")
    }

    func terminalDidClose(processAlive: Bool) {
        // The launcher exited: Esc at the root, or Enter's become finished. The
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
        // A click into another app: the launcher goes away, like any launcher.
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
        Tool.configDir + "/shell-mac.json"
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
