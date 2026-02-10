import SwiftUI
import UIKit

/// UIViewRepresentable that captures touch gestures and maps them to mouse/input events.
/// Coordinate mapping mirrors internal/display/ebiten.go:120-132.
struct TouchInputView: UIViewRepresentable {
    let viewModel: ConnectionViewModel
    let displaySize: CGSize
    let displayOffset: CGPoint
    let frameSize: CGSize

    func makeUIView(context: Context) -> TouchCaptureView {
        let view = TouchCaptureView()
        view.onPan = { [viewModel] location, state in
            let remote = mapToRemote(location)
            switch state {
            case .began:
                viewModel.sendMouseMove(x: remote.x, y: remote.y)
                viewModel.sendMouseDown(x: remote.x, y: remote.y)
            case .changed:
                viewModel.sendMouseMove(x: remote.x, y: remote.y)
            case .ended, .cancelled:
                viewModel.sendMouseUp(x: remote.x, y: remote.y)
            default:
                break
            }
        }
        view.onTap = { [viewModel] location in
            let remote = mapToRemote(location)
            viewModel.sendMouseDown(x: remote.x, y: remote.y)
            viewModel.sendMouseUp(x: remote.x, y: remote.y)
        }
        view.onLongPress = { [viewModel] location in
            let remote = mapToRemote(location)
            viewModel.sendMouseDown(x: remote.x, y: remote.y, button: .right)
            viewModel.sendMouseUp(x: remote.x, y: remote.y, button: .right)
        }
        view.onTwoFingerPan = { [viewModel] translation in
            viewModel.sendScroll(dx: Double(translation.x) * 0.1, dy: Double(-translation.y) * 0.1)
        }
        return view
    }

    func updateUIView(_ uiView: TouchCaptureView, context: Context) {}

    /// Convert view coordinates to remote screen coordinates.
    /// Mirrors ebiten.go:120-132:
    ///   remoteX = (viewX - offsetX) / scale
    ///   remoteY = (viewY - offsetY) / scale
    private func mapToRemote(_ point: CGPoint) -> (x: Double, y: Double) {
        guard displaySize.width > 0, displaySize.height > 0,
              frameSize.width > 0, frameSize.height > 0 else {
            return (0, 0)
        }
        let scaleX = displaySize.width / frameSize.width
        let scaleY = displaySize.height / frameSize.height
        let scale = min(scaleX, scaleY)

        let remoteX = Double(point.x) / scale
        let remoteY = Double(point.y) / scale
        return (remoteX, remoteY)
    }
}

// MARK: - UIKit gesture capture view

final class TouchCaptureView: UIView {
    var onPan: ((CGPoint, UIGestureRecognizer.State) -> Void)?
    var onTap: ((CGPoint) -> Void)?
    var onLongPress: ((CGPoint) -> Void)?
    var onTwoFingerPan: ((CGPoint) -> Void)?

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = .clear
        isMultipleTouchEnabled = true

        let pan = UIPanGestureRecognizer(target: self, action: #selector(handlePan))
        pan.maximumNumberOfTouches = 1
        addGestureRecognizer(pan)

        let tap = UITapGestureRecognizer(target: self, action: #selector(handleTap))
        addGestureRecognizer(tap)

        let longPress = UILongPressGestureRecognizer(target: self, action: #selector(handleLongPress))
        longPress.minimumPressDuration = 0.5
        addGestureRecognizer(longPress)

        let twoFingerPan = UIPanGestureRecognizer(target: self, action: #selector(handleTwoFingerPan))
        twoFingerPan.minimumNumberOfTouches = 2
        twoFingerPan.maximumNumberOfTouches = 2
        addGestureRecognizer(twoFingerPan)

        // Allow tap and long press to work alongside pan
        tap.require(toFail: longPress)
    }

    required init?(coder: NSCoder) { fatalError() }

    @objc private func handlePan(_ gesture: UIPanGestureRecognizer) {
        let location = gesture.location(in: self)
        onPan?(location, gesture.state)
    }

    @objc private func handleTap(_ gesture: UITapGestureRecognizer) {
        let location = gesture.location(in: self)
        onTap?(location)
    }

    @objc private func handleLongPress(_ gesture: UILongPressGestureRecognizer) {
        guard gesture.state == .began else { return }
        let location = gesture.location(in: self)
        onLongPress?(location)
    }

    @objc private func handleTwoFingerPan(_ gesture: UIPanGestureRecognizer) {
        guard gesture.state == .changed else { return }
        let translation = gesture.translation(in: self)
        onTwoFingerPan?(translation)
        gesture.setTranslation(.zero, in: self)
    }
}
