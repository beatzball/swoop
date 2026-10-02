import AppKit
import ApplicationServices

/// Pastes what the launcher asked for, after the panel is off screen.
///
/// The launcher writes a request file (see internal/paste/request.go for
/// the format: a line with the left-arrow count, then the text) and puts
/// the text on the clipboard; the frame, as the launcher leaves, pastes it
/// with cmd+V from its own process. The frame does not know what the text
/// is. It knows: hide, then paste this, if a text field is there.
///
/// This lives in the frame, not the launcher, because the frame owns the
/// moment the panel is gone, and because the Accessibility grant belongs
/// to the process that sends the keys: this one.
enum Paster {
    /// Where the launcher this frame runs writes its request. One per
    /// frame, named by pid, in the per-user temporary directory.
    static let requestPath = NSTemporaryDirectory() + "swoop-paste-\(getpid())"

    /// A request left from before (a launcher that wrote one and then died
    /// before its signal) must not be pasted into whatever comes next.
    static func discard() {
        try? FileManager.default.removeItem(atPath: requestPath)
    }

    /// Read and remove the request, if there is one.
    static func take() -> Request? {
        guard let data = FileManager.default.contents(atPath: requestPath) else { return nil }
        discard()
        return Request(data)
    }

    /// Paste the request into the app in front. Call once the panel is
    /// ordered out.
    static func paste(_ request: Request) {
        // A turn of the run loop, so the window server has moved the
        // keyboard back to the app in front before cmd+V goes out.
        DispatchQueue.main.async { send(request) }
    }

    private static func send(_ request: Request) {
        // The prompt option: the first time, macOS shows its dialog,
        // pointing at swoop-shell-mac. Until the switch is on, copy only.
        // A switch that shows on may hold a grant for an earlier build,
        // which this check cannot tell from none, so the note says both.
        // The key is kAXTrustedCheckOptionPrompt, spelled out: the global
        // is a mutable C variable, which Swift's concurrency checks refuse.
        let options = ["AXTrustedCheckOptionPrompt": true] as CFDictionary
        guard AXIsProcessTrustedWithOptions(options) else {
            notify("Copied \(short(request.text)), not pasted: turn on swoop-shell-mac in System Settings > Privacy & Security > Accessibility, and \(Tool.current.title) pastes for you. Already on? Remove it with the minus button and add it again: a new build needs a new grant.")
            return
        }
        guard focusTakesText() else {
            notify("Copied \(short(request.text)): no text field to paste into.")
            return
        }
        press(9, flags: .maskCommand) // V
        for _ in 0 ..< request.left {
            press(123, flags: []) // left arrow
        }
    }

    /// The roles that take a paste. A web area is a page with no field
    /// focused in it, but also a contenteditable editor, which is where a
    /// lot of typing happens; a search field is a text field by subrole.
    private static let textRoles: Set<String> = ["AXTextField", "AXTextArea", "AXComboBox", "AXWebArea"]

    /// Whether the focused element of the app in front takes text. No
    /// focused element at all counts as yes: terminals and Electron apps
    /// expose none, and cannot be told apart from a field.
    private static func focusTakesText() -> Bool {
        let system = AXUIElementCreateSystemWide()
        var focused: CFTypeRef?
        guard AXUIElementCopyAttributeValue(system, kAXFocusedUIElementAttribute as CFString, &focused) == .success,
              let focused, CFGetTypeID(focused) == AXUIElementGetTypeID()
        else { return true }
        let element = focused as! AXUIElement
        let role = attribute(element, kAXRoleAttribute)
        let subrole = attribute(element, kAXSubroleAttribute)
        return textRoles.contains(role ?? "") || subrole == "AXSearchField"
    }

    private static func attribute(_ element: AXUIElement, _ name: String) -> String? {
        var value: CFTypeRef?
        guard AXUIElementCopyAttributeValue(element, name as CFString, &value) == .success else { return nil }
        return value as? String
    }

    /// One key, down and up, from this process. Events posted to the HID
    /// tap are delivered in order, so the arrows land after the paste.
    private static func press(_ key: CGKeyCode, flags: CGEventFlags) {
        let source = CGEventSource(stateID: .combinedSessionState)
        for down in [true, false] {
            guard let event = CGEvent(keyboardEventSource: source, virtualKey: key, keyDown: down) else { continue }
            // Set even when empty: a key pressed while the user still
            // holds a modifier must not pick it up.
            event.flags = flags
            event.post(tap: .cghidEventTap)
        }
    }

    /// A notification titled with the tool's title, through osascript: a bare binary has
    /// no bundle, and the notification frameworks want one. The text goes
    /// and the title go in as arguments, so no quoting can break them.
    private static func notify(_ note: String) {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        p.arguments = ["-e", "on run argv", "-e", "display notification (item 1 of argv) with title (item 2 of argv)", "-e", "end run", note, Tool.current.title]
        try? p.run()
    }

    /// The text cut to fit a notification: the first line, 40 characters.
    private static func short(_ text: String) -> String {
        let line = text.split(separator: "\n", maxSplits: 1, omittingEmptySubsequences: false).first.map(String.init) ?? ""
        let cut = line.count > 40 || line.count < text.count
        return String(line.prefix(40)) + (cut ? "…" : "")
    }

    /// The request file, parsed: the count line, then the text.
    struct Request {
        let text: String
        let left: Int

        init?(_ data: Data) {
            guard let s = String(data: data, encoding: .utf8),
                  let newline = s.firstIndex(of: "\n"),
                  let n = Int(s[..<newline]), n >= 0
            else { return nil }
            left = n
            text = String(s[s.index(after: newline)...])
        }
    }
}
