import Foundation

// MARK: - Input event types (must match internal/input/events.go)

enum InputEventType: String, Codable {
    case mouseMove = "mouse_move"
    case mouseDown = "mouse_down"
    case mouseUp = "mouse_up"
    case mouseScroll = "mouse_scroll"
    case keyDown = "key_down"
    case keyUp = "key_up"
}

enum MouseButton: Int, Codable {
    case left = 0
    case right = 1
    case middle = 2
}

// Modifier flags bitfield: 1=Shift, 2=Ctrl, 4=Alt/Option, 8=Cmd
struct ModifierFlags: OptionSet {
    let rawValue: UInt8

    static let shift   = ModifierFlags(rawValue: 1)
    static let control = ModifierFlags(rawValue: 2)
    static let alt     = ModifierFlags(rawValue: 4)
    static let command = ModifierFlags(rawValue: 8)
}

// MARK: - Wire format (must match events.go InputEvent struct with omitempty)

struct InputEvent: Codable {
    let type: InputEventType
    var x: Double?
    var y: Double?
    var button: Int?
    var keyCode: UInt16?
    var modifiers: UInt8?
    var scrollDX: Double?
    var scrollDY: Double?

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(type, forKey: .type)
        // Match Go's omitempty: omit zero values
        if let x = x, x != 0 { try container.encode(x, forKey: .x) }
        if let y = y, y != 0 { try container.encode(y, forKey: .y) }
        if let button = button, button != 0 { try container.encode(button, forKey: .button) }
        if let keyCode = keyCode, keyCode != 0 { try container.encode(keyCode, forKey: .keyCode) }
        if let modifiers = modifiers, modifiers != 0 { try container.encode(modifiers, forKey: .modifiers) }
        if let scrollDX = scrollDX, scrollDX != 0 { try container.encode(scrollDX, forKey: .scrollDX) }
        if let scrollDY = scrollDY, scrollDY != 0 { try container.encode(scrollDY, forKey: .scrollDY) }
    }
}

// MARK: - macOS virtual key codes (must match ebiten.go:220-239)

enum MacKeyCode: UInt16 {
    case a = 0x00, s = 0x01, d = 0x02, f = 0x03
    case h = 0x04, g = 0x05, z = 0x06, x = 0x07
    case c = 0x08, v = 0x09, b = 0x0B, q = 0x0C
    case w = 0x0D, e = 0x0E, r = 0x0F, y = 0x10
    case t = 0x11
    case key1 = 0x12, key2 = 0x13, key3 = 0x14, key4 = 0x15
    case key6 = 0x16, key5 = 0x17, key9 = 0x19, key7 = 0x1A
    case key8 = 0x1C, key0 = 0x1D
    case o = 0x1F, u = 0x20, i = 0x22, p = 0x23
    case l = 0x25, j = 0x26, k = 0x28
    case n = 0x2D, m = 0x2E
    case returnKey = 0x24
    case tab = 0x30
    case space = 0x31
    case delete = 0x33
    case escape = 0x35
    case leftArrow = 0x7B, rightArrow = 0x7C
    case downArrow = 0x7D, upArrow = 0x7E
    case f1 = 0x7A, f2 = 0x78, f3 = 0x63, f4 = 0x76
    case f5 = 0x60, f6 = 0x61, f7 = 0x62, f8 = 0x64
    case f9 = 0x65, f10 = 0x6D, f11 = 0x67, f12 = 0x6F
    case forwardDelete = 0x75, home = 0x73, end = 0x77
    case pageUp = 0x74, pageDown = 0x79
    case unmapped = 0xFF
}
