import Foundation
import WebRTC
import os

private let logger = Logger(subsystem: "com.airmac.ios", category: "WebRTCClient")

protocol WebRTCClientDelegate: AnyObject {
    func webRTCDidReceiveFrame(_ data: Data)
    func webRTCDidGenerateICECandidate(_ candidate: RTCIceCandidate)
    func webRTCDidChangeConnectionState(_ state: RTCPeerConnectionState)
    func webRTCDidOpenDataChannel(label: String)
}

/// WebRTC peer connection manager — mirrors internal/peer/controller.go
/// Controller creates offer, does NOT create data channels (host creates them).
final class WebRTCClient: NSObject {
    private static let factory: RTCPeerConnectionFactory = {
        RTCInitializeSSL()
        return RTCPeerConnectionFactory()
    }()

    private var peerConnection: RTCPeerConnection?
    private var framesChannel: RTCDataChannel?
    private var inputChannel: RTCDataChannel?

    weak var delegate: WebRTCClientDelegate?

    override init() {
        super.init()
    }

    // MARK: - Setup (mirrors NewPeerConnection + NewController)

    func setup() {
        let config = RTCConfiguration()
        // ICE servers matching peer.go:ICEServers
        config.iceServers = [
            RTCIceServer(urlStrings: [
                "stun:stun.l.google.com:19302",
                "stun:stun1.l.google.com:19302"
            ])
        ]
        config.sdpSemantics = .unifiedPlan
        config.continualGatheringPolicy = .gatherContinually

        let constraints = RTCMediaConstraints(
            mandatoryConstraints: nil,
            optionalConstraints: ["DtlsSrtpKeyAgreement": "true"]
        )

        peerConnection = WebRTCClient.factory.peerConnection(
            with: config,
            constraints: constraints,
            delegate: self
        )

        logger.info("PeerConnection created")
    }

    // MARK: - Connect (mirrors Controller.Connect)

    /// Creates an offer SDP and returns it as a dictionary matching pion/webrtc format:
    /// {"type":"offer","sdp":"v=0\r\n..."}
    func createOffer() async throws -> [String: Any] {
        guard let pc = peerConnection else { throw WebRTCError.notSetup }

        let constraints = RTCMediaConstraints(
            mandatoryConstraints: [
                "OfferToReceiveAudio": "false",
                "OfferToReceiveVideo": "false"
            ],
            optionalConstraints: nil
        )

        let sdp = try await pc.offer(for: constraints)
        try await pc.setLocalDescription(sdp)

        // Format must match pion/webrtc JSON: {"type":"offer","sdp":"..."}
        return [
            "type": sdpTypeToString(sdp.type),
            "sdp": sdp.sdp
        ]
    }

    // MARK: - Handle Answer (mirrors Controller.HandleAnswer)

    func handleAnswer(_ payload: [String: Any]) throws {
        guard let pc = peerConnection else { throw WebRTCError.notSetup }
        guard let typeStr = payload["type"] as? String,
              let sdpStr = payload["sdp"] as? String else {
            throw WebRTCError.invalidSDP
        }

        let type = stringToSdpType(typeStr)
        let sdp = RTCSessionDescription(type: type, sdp: sdpStr)
        pc.setRemoteDescription(sdp) { error in
            if let error = error {
                logger.error("setRemoteDescription error: \(error.localizedDescription)")
            } else {
                logger.info("Remote description set (answer)")
            }
        }
    }

    // MARK: - Handle ICE Candidate (mirrors Controller.HandleICECandidate)

    func handleICECandidate(_ payload: [String: Any]) throws {
        guard let pc = peerConnection else { throw WebRTCError.notSetup }
        // Format: {"candidate":"...","sdpMLineIndex":0,"sdpMid":"0"}
        guard let candidateStr = payload["candidate"] as? String else {
            throw WebRTCError.invalidCandidate
        }

        let sdpMLineIndex = (payload["sdpMLineIndex"] as? Int32) ?? 0
        let sdpMid = payload["sdpMid"] as? String

        let candidate = RTCIceCandidate(
            sdp: candidateStr,
            sdpMLineIndex: sdpMLineIndex,
            sdpMid: sdpMid
        )

        pc.add(candidate) { error in
            if let error = error {
                logger.error("addICECandidate error: \(error.localizedDescription)")
            }
        }
    }

    // MARK: - Send Input (mirrors transport.SendInput)

    func sendInput(_ data: Data) {
        guard let channel = inputChannel,
              channel.readyState == .open else { return }
        let buffer = RTCDataBuffer(data: data, isBinary: false)
        channel.sendData(buffer)
    }

    // MARK: - Close (mirrors Controller.Close)

    func close() {
        framesChannel?.close()
        inputChannel?.close()
        peerConnection?.close()
        framesChannel = nil
        inputChannel = nil
        peerConnection = nil
    }

    // MARK: - SDP type helpers

    private func sdpTypeToString(_ type: RTCSdpType) -> String {
        switch type {
        case .offer: return "offer"
        case .prAnswer: return "pranswer"
        case .answer: return "answer"
        case .rollback: return "rollback"
        @unknown default: return "offer"
        }
    }

    private func stringToSdpType(_ string: String) -> RTCSdpType {
        switch string {
        case "offer": return .offer
        case "answer": return .answer
        case "pranswer": return .prAnswer
        case "rollback": return .rollback
        default: return .offer
        }
    }
}

// MARK: - RTCPeerConnectionDelegate

extension WebRTCClient: RTCPeerConnectionDelegate {
    func peerConnection(_ peerConnection: RTCPeerConnection, didChange stateChanged: RTCSignalingState) {
        logger.info("Signaling state: \(stateChanged.rawValue)")
    }

    func peerConnection(_ peerConnection: RTCPeerConnection, didAdd stream: RTCMediaStream) {}
    func peerConnection(_ peerConnection: RTCPeerConnection, didRemove stream: RTCMediaStream) {}

    func peerConnectionShouldNegotiate(_ peerConnection: RTCPeerConnection) {
        logger.info("Negotiation needed")
    }

    func peerConnection(_ peerConnection: RTCPeerConnection, didChange newState: RTCIceConnectionState) {
        logger.info("ICE connection state: \(newState.rawValue)")
    }

    func peerConnection(_ peerConnection: RTCPeerConnection, didChange newState: RTCIceGatheringState) {
        logger.info("ICE gathering state: \(newState.rawValue)")
    }

    func peerConnection(_ peerConnection: RTCPeerConnection, didChange newState: RTCPeerConnectionState) {
        logger.info("Peer connection state: \(newState.rawValue)")
        DispatchQueue.main.async {
            self.delegate?.webRTCDidChangeConnectionState(newState)
        }
    }

    /// ICE candidate generated — send to remote via signaling
    /// Format: {"candidate":"...","sdpMLineIndex":0,"sdpMid":"0"}
    func peerConnection(_ peerConnection: RTCPeerConnection, didGenerate candidate: RTCIceCandidate) {
        logger.info("ICE candidate generated")
        DispatchQueue.main.async {
            self.delegate?.webRTCDidGenerateICECandidate(candidate)
        }
    }

    func peerConnection(_ peerConnection: RTCPeerConnection, didRemove candidates: [RTCIceCandidate]) {}

    /// Accept data channels from host (mirrors controller.go:OnDataChannel)
    func peerConnection(_ peerConnection: RTCPeerConnection, didOpen dataChannel: RTCDataChannel) {
        logger.info("Data channel received: \(dataChannel.label)")
        switch dataChannel.label {
        case "frames":
            framesChannel = dataChannel
            dataChannel.delegate = self
        case "input":
            inputChannel = dataChannel
            dataChannel.delegate = self
        default:
            logger.warning("Unknown data channel: \(dataChannel.label)")
        }
        DispatchQueue.main.async {
            self.delegate?.webRTCDidOpenDataChannel(label: dataChannel.label)
        }
    }
}

// MARK: - RTCDataChannelDelegate

extension WebRTCClient: RTCDataChannelDelegate {
    func dataChannelDidChangeState(_ dataChannel: RTCDataChannel) {
        logger.info("Data channel '\(dataChannel.label)' state: \(dataChannel.readyState.rawValue)")
    }

    func dataChannel(_ dataChannel: RTCDataChannel, didReceiveMessageWith buffer: RTCDataBuffer) {
        if dataChannel.label == "frames" {
            // Raw JPEG bytes — deliver on background to avoid blocking
            let data = buffer.data
            DispatchQueue.main.async {
                self.delegate?.webRTCDidReceiveFrame(data)
            }
        }
    }
}

// MARK: - Errors

enum WebRTCError: Error {
    case notSetup
    case invalidSDP
    case invalidCandidate
}
