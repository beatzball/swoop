import Carbon

/// A global hotkey through Carbon's RegisterEventHotKey. It works from an
/// accessory app with no permissions: the system delivers the key to this
/// process no matter which app is in front. The one limit is that a key
/// another app already registered cannot be taken; registration then fails
/// and the error says so.
final class HotKey {
    struct ParseError: Error, CustomStringConvertible {
        let description: String
    }

    private var ref: EventHotKeyRef?
    private var handler: EventHandlerRef?
    private let action: () -> Void

    /// spec is "mod+mod+key", such as "alt+shift+space" or "alt+space".
    init(_ spec: String, action: @escaping () -> Void) throws {
        self.action = action
        let (keyCode, modifiers) = try Self.parse(spec)

        var eventType = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        let selfPtr = Unmanaged.passUnretained(self).toOpaque()
        let status = InstallEventHandler(GetApplicationEventTarget(), { _, _, userData in
            guard let userData else { return noErr }
            Unmanaged<HotKey>.fromOpaque(userData).takeUnretainedValue().action()
            return noErr
        }, 1, &eventType, selfPtr, &handler)
        guard status == noErr else {
            throw ParseError(description: "cannot install the hotkey handler (\(status))")
        }

        let id = EventHotKeyID(signature: 0x5357_4F50 /* SWOP */, id: 1)
        let reg = RegisterEventHotKey(keyCode, modifiers, id, GetApplicationEventTarget(), 0, &ref)
        guard reg == noErr else {
            throw ParseError(description: "cannot register hotkey \(spec) (\(reg)); is another app using it?")
        }
    }

    deinit {
        if let ref { UnregisterEventHotKey(ref) }
        if let handler { RemoveEventHandler(handler) }
    }

    private static let keys: [String: Int] = [
        "space": kVK_Space, "return": kVK_Return, "enter": kVK_Return, "tab": kVK_Tab,
        "escape": kVK_Escape, "esc": kVK_Escape, "`": kVK_ANSI_Grave, "grave": kVK_ANSI_Grave,
        "a": kVK_ANSI_A, "b": kVK_ANSI_B, "c": kVK_ANSI_C, "d": kVK_ANSI_D, "e": kVK_ANSI_E,
        "f": kVK_ANSI_F, "g": kVK_ANSI_G, "h": kVK_ANSI_H, "i": kVK_ANSI_I, "j": kVK_ANSI_J,
        "k": kVK_ANSI_K, "l": kVK_ANSI_L, "m": kVK_ANSI_M, "n": kVK_ANSI_N, "o": kVK_ANSI_O,
        "p": kVK_ANSI_P, "q": kVK_ANSI_Q, "r": kVK_ANSI_R, "s": kVK_ANSI_S, "t": kVK_ANSI_T,
        "u": kVK_ANSI_U, "v": kVK_ANSI_V, "w": kVK_ANSI_W, "x": kVK_ANSI_X, "y": kVK_ANSI_Y,
        "z": kVK_ANSI_Z,
    ]

    private static func parse(_ spec: String) throws -> (UInt32, UInt32) {
        var modifiers: UInt32 = 0
        var keyCode: Int?
        for part in spec.lowercased().split(separator: "+") {
            switch part {
            case "cmd", "command", "super": modifiers |= UInt32(cmdKey)
            case "shift": modifiers |= UInt32(shiftKey)
            case "alt", "opt", "option": modifiers |= UInt32(optionKey)
            case "ctrl", "control": modifiers |= UInt32(controlKey)
            default:
                guard let code = keys[String(part)] else {
                    throw ParseError(description: "unknown key \"\(part)\" in hotkey \"\(spec)\"")
                }
                keyCode = code
            }
        }
        guard let keyCode else {
            throw ParseError(description: "no key in hotkey \"\(spec)\"")
        }
        return (UInt32(keyCode), modifiers)
    }
}
