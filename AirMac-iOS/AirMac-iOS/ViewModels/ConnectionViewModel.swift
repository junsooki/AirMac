import Foundation
import SwiftUI
import WebRTC
import os

private let logger = Logger(subsystem: "com.airmac.ios", category: "ConnectionVM")

// MARK: - Connection state machine

enum ConnectionState: Equatable {
    case disconnected
    case connecting
    case registered
    case selectingHost
    case connected
}

/// Orchestrates signaling + WebRTC — mirrors cmd/controller/main.go
@MainActor
final class ConnectionViewModel: ObservableObject {
    @Published var state: ConnectionState = .disconnected
    @Published var hosts: [HostInfo] = []
    @Published var currentFrame: UIImage?
    @Published var errorMessage: String?
    @Published var connectedHostID: String?

    // Frame dimensions for coordinate mapping
    @Published var frameWidth: CGFloat = 0
    @Published var frameHeight: CGFloat = 0

    private var signalingClient: SignalingClient?
    private var webRTCClient: WebRTCClient?
    private var targetHostID: String?

    // MARK: - Connect to signaling server

    func connect(signalingURL: String) {
        guard state == .disconnected else { return }
        state = .connecting
        errorMessage = nil

        let client = SignalingClient()
        client.delegate = self
        signalingClient = client
        client.connect(to: signalingURL)
    }

    // MARK: - Select host and establish WebRTC

    func selectHost(_ hostID: String) {
        guard state == .selectingHost else { return }
        targetHostID = hostID
        connectedHostID = hostID

        // Create WebRTC peer (mirrors NewController in controller.go)
        let rtc = WebRTCClient()
        rtc.delegate = self
        rtc.setup()
        webRTCClient = rtc

        // Create and send offer (mirrors Controller.Connect)
        Task {
            do {
                let offer = try await rtc.createOffer()
                signalingClient?.sendOffer(target: hostID, payload: offer)
                logger.info("Offer sent to \(hostID)")
            } catch {
                logger.error("Failed to create offer: \(error.localizedDescription)")
                self.errorMessage = "Failed to create offer: \(error.localizedDescription)"
            }
        }
    }

    func refreshHostList() {
        signalingClient?.requestHostList()
    }

    // MARK: - Disconnect

    func disconnect() {
        webRTCClient?.close()
        webRTCClient = nil
        signalingClient?.disconnect()
        signalingClient = nil
        targetHostID = nil
        connectedHostID = nil
        currentFrame = nil
        frameWidth = 0
        frameHeight = 0
        hosts = []
        state = .disconnected
    }

    // MARK: - Input sending (mirrors display/ebiten.go:sendInput)

    func sendMouseMove(x: Double, y: Double) {
        sendInputEvent(InputEvent(type: .mouseMove, x: x, y: y))
    }

    func sendMouseDown(x: Double, y: Double, button: MouseButton = .left) {
        sendInputEvent(InputEvent(type: .mouseDown, x: x, y: y, button: button.rawValue))
    }

    func sendMouseUp(x: Double, y: Double, button: MouseButton = .left) {
        sendInputEvent(InputEvent(type: .mouseUp, x: x, y: y, button: button.rawValue))
    }

    func sendScroll(dx: Double, dy: Double) {
        sendInputEvent(InputEvent(type: .mouseScroll, scrollDX: dx, scrollDY: dy))
    }

    func sendKeyDown(keyCode: UInt16, modifiers: UInt8 = 0) {
        sendInputEvent(InputEvent(type: .keyDown, keyCode: keyCode, modifiers: modifiers))
    }

    func sendKeyUp(keyCode: UInt16, modifiers: UInt8 = 0) {
        sendInputEvent(InputEvent(type: .keyUp, keyCode: keyCode, modifiers: modifiers))
    }

    private func sendInputEvent(_ event: InputEvent) {
        guard let data = try? JSONEncoder().encode(event) else { return }
        webRTCClient?.sendInput(data)
    }
}

// MARK: - SignalingClientDelegate

extension ConnectionViewModel: SignalingClientDelegate {
    func signalingDidRegister() {
        logger.info("Registered — requesting host list")
        state = .selectingHost
        signalingClient?.requestHostList()
    }

    func signalingDidReceiveHosts(_ hosts: [HostInfo]) {
        self.hosts = hosts
        logger.info("Received \(hosts.count) host(s)")
    }

    // mirrors cmd/controller/main.go:OnAnswer
    func signalingDidReceiveAnswer(from: String, payload: [String: Any]) {
        do {
            try webRTCClient?.handleAnswer(payload)
            logger.info("Answer handled from \(from)")
        } catch {
            logger.error("Handle answer: \(error.localizedDescription)")
        }
    }

    // mirrors cmd/controller/main.go:OnICECandidate
    func signalingDidReceiveICECandidate(from: String, payload: [String: Any]) {
        do {
            try webRTCClient?.handleICECandidate(payload)
        } catch {
            logger.error("Handle ICE candidate: \(error.localizedDescription)")
        }
    }

    func signalingDidReceiveHostDisconnected(hostID: String) {
        if hostID == targetHostID {
            logger.info("Host disconnected: \(hostID)")
            errorMessage = "Host disconnected"
            disconnect()
        }
        hosts.removeAll { $0.id == hostID }
    }

    func signalingDidReceiveError(message: String) {
        errorMessage = message
    }

    func signalingDidDisconnect() {
        if state != .disconnected {
            errorMessage = "Connection lost"
            disconnect()
        }
    }
}

// MARK: - WebRTCClientDelegate

extension ConnectionViewModel: WebRTCClientDelegate {
    func webRTCDidReceiveFrame(_ data: Data) {
        // Decode JPEG on background thread (mirrors decoder.NewJPEGDecoder + SetFrame)
        Task.detached(priority: .userInitiated) {
            guard let image = UIImage(data: data) else { return }
            await MainActor.run {
                self.currentFrame = image
                self.frameWidth = image.size.width
                self.frameHeight = image.size.height
            }
        }
    }

    /// Send ICE candidate to remote via signaling (mirrors controller.go:OnICECandidate)
    func webRTCDidGenerateICECandidate(_ candidate: RTCIceCandidate) {
        guard let hostID = targetHostID else { return }
        // Format: {"candidate":"...","sdpMLineIndex":0,"sdpMid":"0"}
        let payload: [String: Any] = [
            "candidate": candidate.sdp,
            "sdpMLineIndex": candidate.sdpMLineIndex,
            "sdpMid": candidate.sdpMid ?? "0"
        ]
        signalingClient?.sendICECandidate(target: hostID, payload: payload)
    }

    func webRTCDidChangeConnectionState(_ state: RTCPeerConnectionState) {
        switch state {
        case .connected:
            self.state = .connected
            logger.info("WebRTC connected")
        case .disconnected, .failed:
            logger.info("WebRTC disconnected/failed")
            errorMessage = "Peer connection \(state == .failed ? "failed" : "disconnected")"
            disconnect()
        default:
            break
        }
    }

    func webRTCDidOpenDataChannel(label: String) {
        logger.info("Data channel opened: \(label)")
    }
}
