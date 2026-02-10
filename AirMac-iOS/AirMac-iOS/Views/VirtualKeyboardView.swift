import SwiftUI

struct VirtualKeyboardView: View {
    let viewModel: ConnectionViewModel
    @State private var shiftOn = false
    @State private var ctrlOn = false
    @State private var altOn = false
    @State private var cmdOn = false

    var body: some View {
        VStack(spacing: 8) {
            // Modifier toggles
            HStack(spacing: 12) {
                modifierToggle("Shift", isOn: $shiftOn)
                modifierToggle("Ctrl", isOn: $ctrlOn)
                modifierToggle("Alt", isOn: $altOn)
                modifierToggle("Cmd", isOn: $cmdOn)
            }
            .padding(.top, 12)

            // Row 1: Numbers
            keyRow([
                ("Esc", MacKeyCode.escape),
                ("1", .key1), ("2", .key2), ("3", .key3), ("4", .key4), ("5", .key5),
                ("6", .key6), ("7", .key7), ("8", .key8), ("9", .key9), ("0", .key0),
            ])

            // Row 2: QWERTY top
            keyRow([
                ("Tab", .tab),
                ("Q", .q), ("W", .w), ("E", .e), ("R", .r), ("T", .t),
                ("Y", .y), ("U", .u), ("I", .i), ("O", .o), ("P", .p),
            ])

            // Row 3: Home row
            keyRow([
                ("A", .a), ("S", .s), ("D", .d), ("F", .f), ("G", .g),
                ("H", .h), ("J", .j), ("K", .k), ("L", .l),
                ("Ret", .returnKey),
            ])

            // Row 4: Bottom row
            keyRow([
                ("Z", .z), ("X", .x), ("C", .c), ("V", .v), ("B", .b),
                ("N", .n), ("M", .m), ("Del", .delete),
            ])

            // Row 5: Space + arrows
            HStack(spacing: 4) {
                keyButton("Space", keyCode: .space, flex: true)
                keyButton("\u{2190}", keyCode: .leftArrow)
                keyButton("\u{2193}", keyCode: .downArrow)
                keyButton("\u{2191}", keyCode: .upArrow)
                keyButton("\u{2192}", keyCode: .rightArrow)
            }
            .padding(.horizontal, 8)

            // Row 6: Function keys
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 4) {
                    ForEach(1...12, id: \.self) { n in
                        let code = fKeyCode(n)
                        keyButton("F\(n)", keyCode: code)
                    }
                }
                .padding(.horizontal, 8)
            }
            .padding(.bottom, 8)
        }
        .background(Color(.systemGroupedBackground))
    }

    // MARK: - Components

    private func modifierToggle(_ label: String, isOn: Binding<Bool>) -> some View {
        Button {
            isOn.wrappedValue.toggle()
        } label: {
            Text(label)
                .font(.caption.bold())
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(isOn.wrappedValue ? Color.accentColor : Color(.systemGray5))
                .foregroundColor(isOn.wrappedValue ? .white : .primary)
                .cornerRadius(8)
        }
    }

    private func keyRow(_ keys: [(String, MacKeyCode)]) -> some View {
        HStack(spacing: 4) {
            ForEach(keys, id: \.1) { label, code in
                keyButton(label, keyCode: code)
            }
        }
        .padding(.horizontal, 8)
    }

    private func keyButton(_ label: String, keyCode: MacKeyCode, flex: Bool = false) -> some View {
        Button {
            let mods = currentModifiers()
            viewModel.sendKeyDown(keyCode: keyCode.rawValue, modifiers: mods)
            viewModel.sendKeyUp(keyCode: keyCode.rawValue, modifiers: mods)
        } label: {
            Text(label)
                .font(.system(size: 14, weight: .medium, design: .monospaced))
                .frame(maxWidth: flex ? .infinity : nil, minWidth: flex ? nil : 28, minHeight: 36)
                .padding(.horizontal, 4)
                .background(Color(.systemBackground))
                .cornerRadius(6)
                .shadow(color: .black.opacity(0.1), radius: 1, y: 1)
        }
        .buttonStyle(.plain)
    }

    private func currentModifiers() -> UInt8 {
        var m: UInt8 = 0
        if shiftOn { m |= ModifierFlags.shift.rawValue }
        if ctrlOn { m |= ModifierFlags.control.rawValue }
        if altOn { m |= ModifierFlags.alt.rawValue }
        if cmdOn { m |= ModifierFlags.command.rawValue }
        return m
    }

    private func fKeyCode(_ n: Int) -> MacKeyCode {
        switch n {
        case 1: return .f1; case 2: return .f2; case 3: return .f3; case 4: return .f4
        case 5: return .f5; case 6: return .f6; case 7: return .f7; case 8: return .f8
        case 9: return .f9; case 10: return .f10; case 11: return .f11; case 12: return .f12
        default: return .unmapped
        }
    }
}
