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
/// One swoop per show, and the next one is started as soon as the current
/// one is done, while the panel is hidden. So a press never shows an empty
/// terminal waiting for fzf: the launcher behind the panel is already drawn.
/// When swoop exits, or the panel loses the keyboard, the panel hides at
/// once and the surface is replaced. That is the answer to "a fresh
/// launcher on every open": the frame knows when it hides, and Ghostty's
/// quick terminal did not.
final class LauncherController: NSObject, NSWindowDelegate,
    TerminalSurfaceCloseDelegate, TerminalSurfaceCommandFinishedDelegate
{
    private let command: String
    private let panel: LauncherPanel
    private var terminal: TerminalView?
    private var settings = Settings.load()
    private lazy var controller = TerminalController(configuration: configuration())
    private var keyMonitor: Any?

    private static let size = NSSize(width: 880, height: 520)

    init(command: String) {
        self.command = command
        panel = LauncherPanel(
            contentRect: NSRect(origin: .zero, size: Self.size),
            styleMask: [.borderless, .nonactivatingPanel, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        super.init()
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

        let content = NSView(frame: NSRect(origin: .zero, size: Self.size))
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
            // The frame's own name, for a launcher that wants to know.
            envVars: ["SWOOP_SHELL": "mac"],
            command: command,
            waitAfterCommand: false
        )
        view.controller = controller
        content.addSubview(view)
        terminal = view
    }

    func show() {
        if terminal == nil { prepare() }
        centerOnActiveScreen()
        panel.makeKeyAndOrderFront(nil)
        if let terminal { panel.makeFirstResponder(terminal) }
    }

    func hide() {
        panel.orderOut(nil)
        // Dropping the view closes its surface, which ends the process if
        // it is still running. Then the next one starts right away, hidden,
        // so the next show is instant.
        terminal?.removeFromSuperview()
        terminal = nil
        prepare()
    }

    private func centerOnActiveScreen() {
        // The screen with the mouse: where the user is looking.
        let mouse = NSEvent.mouseLocation
        let screen = NSScreen.screens.first { $0.frame.contains(mouse) } ?? NSScreen.main
        guard let frame = screen?.visibleFrame else { return }
        let origin = NSPoint(
            x: frame.midX - Self.size.width / 2,
            // A little above centre, like Spotlight.
            y: frame.midY - Self.size.height / 2 + frame.height * 0.08
        )
        panel.setFrame(NSRect(origin: origin, size: Self.size), display: true)
    }

    // MARK: the surface says it is done

    func terminalDidFinishCommand(exitCode _: Int?, durationNanos _: UInt64) {
        // swoop exited: Esc at the root, or Enter's become finished. Hide
        // now, before the terminal can show "Process exited".
        DispatchQueue.main.async { if self.panel.isVisible { self.hide() } }
    }

    func terminalDidClose(processAlive _: Bool) {
        DispatchQueue.main.async { if self.panel.isVisible { self.hide() } }
    }

    // MARK: the panel lost the keyboard

    func windowDidResignKey(_: Notification) {
        // A click into another app: the launcher goes away, like Raycast.
        if panel.isVisible { hide() }
    }
}

/// What the frame remembers between runs: the font size. A small JSON file
/// under the config directory, next to the extensions.
struct Settings: Codable {
    static let defaultFontSize: Float = 16
    var fontSize: Float = Settings.defaultFontSize

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
