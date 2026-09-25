import AppKit
import GhosttyTerminal

/// A borderless floating panel that can take the keyboard without making
/// this app the front app, which is what a launcher wants: it appears over
/// whatever you were doing and leaves it in place.
final class LauncherPanel: NSPanel {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }
}

/// Shows and hides the panel and owns the terminal surface inside it. One
/// surface per show: when swoop exits, or the panel loses the keyboard,
/// the surface is dropped, and the next show makes a new one. That is the
/// answer to "a fresh launcher on every open": the frame knows when it
/// hides, and Ghostty's quick terminal did not.
final class LauncherController: NSObject, NSWindowDelegate, TerminalSurfaceCloseDelegate {
    private let command: String
    private let panel: LauncherPanel
    private var terminal: TerminalView?
    private lazy var controller = TerminalController { builder in
        builder.withFontSize(16)
        builder.withBackgroundOpacity(0.96)
        builder.withWindowPaddingX(12)
        builder.withWindowPaddingY(8)
        // The cursor is fzf's; a blinking block would fight with it.
        builder.withCursorStyleBlink(false)
    }

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
    }

    func toggle() {
        if panel.isVisible { hide() } else { show() }
    }

    func show() {
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

        centerOnActiveScreen()
        panel.makeKeyAndOrderFront(nil)
        panel.makeFirstResponder(view)
    }

    func hide() {
        panel.orderOut(nil)
        // Dropping the view closes its surface, which ends the process if
        // it is still running. The next show starts clean.
        terminal?.removeFromSuperview()
        terminal = nil
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

    func terminalDidClose(processAlive _: Bool) {
        // swoop exited: Esc at the root, or Enter's become finished.
        DispatchQueue.main.async { self.hide() }
    }

    // MARK: the panel lost the keyboard

    func windowDidResignKey(_: Notification) {
        // A click into another app: the launcher goes away, like Raycast.
        hide()
    }
}
