import AppKit

/// A hint that the panel's edges drag. It sits above the terminal, lets
/// every click through, and watches the mouse: within `reach` points of
/// an edge it draws a short bar on that edge, centred where the mouse
/// is; within reach of two edges, a small mark on the corner. Anywhere
/// else, nothing. The resize itself is macOS's own edge drag; this only
/// says where it is.
final class EdgeHintView: NSView {
    /// How close to an edge, in points, before the hint shows.
    private let reach: CGFloat = 12
    private let barLength: CGFloat = 44
    private let barThickness: CGFloat = 3
    private let inset: CGFloat = 3

    private struct Hint: Equatable {
        var left = false, right = false, top = false, bottom = false
        var point = NSPoint.zero
        var any: Bool { left || right || top || bottom }
    }

    private var hint = Hint() {
        didSet { if hint != oldValue { needsDisplay = true } }
    }

    override init(frame: NSRect) {
        super.init(frame: frame)
        wantsLayer = true
    }

    required init?(coder: NSCoder) { nil }

    /// Never the target of a click: the terminal below gets them all.
    override func hitTest(_: NSPoint) -> NSView? { nil }

    override func updateTrackingAreas() {
        super.updateTrackingAreas()
        for area in trackingAreas { removeTrackingArea(area) }
        addTrackingArea(NSTrackingArea(
            rect: bounds,
            options: [.mouseMoved, .mouseEnteredAndExited, .activeAlways, .inVisibleRect],
            owner: self,
            userInfo: nil
        ))
    }

    override func mouseMoved(with event: NSEvent) {
        let p = convert(event.locationInWindow, from: nil)
        var h = Hint(point: p)
        h.left = p.x < reach
        h.right = p.x > bounds.width - reach
        h.bottom = p.y < reach
        h.top = p.y > bounds.height - reach
        hint = h
    }

    override func mouseExited(with _: NSEvent) {
        hint = Hint()
    }

    override func draw(_: NSRect) {
        guard hint.any else { return }
        NSColor.white.withAlphaComponent(0.55).setFill()
        let corner = (hint.left || hint.right) && (hint.top || hint.bottom)
        if corner {
            // A small rounded square tucked into the corner.
            let s: CGFloat = 8
            let x = hint.left ? inset : bounds.width - inset - s
            let y = hint.bottom ? inset : bounds.height - inset - s
            NSBezierPath(roundedRect: NSRect(x: x, y: y, width: s, height: s), xRadius: 2, yRadius: 2).fill()
            return
        }
        // A bar along the edge, centred on the mouse, kept inside the
        // panel's rounded corners.
        var rect: NSRect
        if hint.left || hint.right {
            let y = min(max(hint.point.y - barLength / 2, 14), bounds.height - 14 - barLength)
            let x = hint.left ? inset : bounds.width - inset - barThickness
            rect = NSRect(x: x, y: y, width: barThickness, height: barLength)
        } else {
            let x = min(max(hint.point.x - barLength / 2, 14), bounds.width - 14 - barLength)
            let y = hint.bottom ? inset : bounds.height - inset - barThickness
            rect = NSRect(x: x, y: y, width: barLength, height: barThickness)
        }
        NSBezierPath(roundedRect: rect, xRadius: barThickness / 2, yRadius: barThickness / 2).fill()
    }
}
