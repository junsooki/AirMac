import Foundation
import os

private let logger = Logger(subsystem: "com.airmac.ios", category: "SignalingClient")

protocol SignalingClientDelegate: AnyObject {
    func signalingDidRegister()
    func signalingDidReceiveHosts(_ hosts: [HostInfo])
    func signalingDidReceiveAnswer(from: String, payload: [String: Any])
    func signalingDidReceiveICECandidate(from: String, payload: [String: Any])
    func signalingDidReceiveHostDisconnected(hostID: String)
    func signalingDidReceiveError(message: String)
    func signalingDidDisconnect()
}

/// WebSocket signaling client — mirrors internal/signaling/client.go
final class SignalingClient {
    private let clientID: String
    private let clientType: ClientType = .controller
    private var webSocket: URLSessionWebSocketTask?
    private var session: URLSession?
    private var pingTimer: Timer?
    private var isClosed = false

    weak var delegate: SignalingClientDelegate?

    init(clientID: String = "controller-\(UUID().uuidString.prefix(8).lowercased())") {
        self.clientID = clientID
    }

    // MARK: - Connect / Disconnect

    func connect(to urlString: String) {
        guard let url = URL(string: urlString) else {
            logger.error("Invalid signaling URL: \(urlString)")
            return
        }

        isClosed = false
        session = URLSession(configuration: .default)
        webSocket = session?.webSocketTask(with: url)
        webSocket?.resume()

        // Register immediately after connecting (matches client.go:Connect)
        let registerMsg = SignalingMessage(
            type: .register,
            id: clientID,
            clientType: clientType
        )
        send(registerMsg)

        receiveNext()
        startPingTimer()

        logger.info("Connecting to \(urlString) as \(self.clientID)")
    }

    func disconnect() {
        isClosed = true
        pingTimer?.invalidate()
        pingTimer = nil
        webSocket?.cancel(with: .goingAway, reason: nil)
        webSocket = nil
        session?.invalidateAndCancel()
        session = nil
    }

    // MARK: - Send methods (match client.go: SendOffer, SendAnswer, SendICECandidate, RequestHostList)

    func requestHostList() {
        send(SignalingMessage(type: .listHosts))
    }

    func sendOffer(target: String, payload: [String: Any]) {
        send(SignalingMessage(
            type: .offer,
            target: target,
            payload: AnyCodable(payload)
        ))
    }

    func sendAnswer(target: String, payload: [String: Any]) {
        send(SignalingMessage(
            type: .answer,
            target: target,
            payload: AnyCodable(payload)
        ))
    }

    func sendICECandidate(target: String, payload: [String: Any]) {
        send(SignalingMessage(
            type: .iceCandidate,
            target: target,
            payload: AnyCodable(payload)
        ))
    }

    // MARK: - Private

    private func send(_ message: SignalingMessage) {
        guard let webSocket = webSocket else { return }
        do {
            let data = try JSONEncoder().encode(message)
            webSocket.send(.data(data)) { error in
                if let error = error {
                    logger.error("Send error: \(error.localizedDescription)")
                }
            }
        } catch {
            logger.error("Encode error: \(error.localizedDescription)")
        }
    }

    /// Recursive receive loop (matches client.go:readLoop)
    private func receiveNext() {
        guard !isClosed else { return }
        webSocket?.receive { [weak self] result in
            guard let self = self, !self.isClosed else { return }
            switch result {
            case .success(let message):
                self.handleMessage(message)
                self.receiveNext()
            case .failure(let error):
                if !self.isClosed {
                    logger.error("Receive error: \(error.localizedDescription)")
                    DispatchQueue.main.async {
                        self.delegate?.signalingDidDisconnect()
                    }
                }
            }
        }
    }

    private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case .data(let d):
            data = d
        case .string(let s):
            guard let d = s.data(using: .utf8) else { return }
            data = d
        @unknown default:
            return
        }

        // Parse as dictionary to preserve raw payload
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let typeStr = json["type"] as? String else {
            return
        }

        dispatch(type: typeStr, json: json)
    }

    /// Message dispatch (matches client.go:dispatch)
    private func dispatch(type: String, json: [String: Any]) {
        DispatchQueue.main.async { [weak self] in
            guard let self = self else { return }
            switch type {
            case "registered":
                logger.info("Registered with signaling server")
                self.delegate?.signalingDidRegister()

            case "hosts", "hosts-updated":
                if let listData = json["list"] as? [[String: Any]] {
                    let hosts = listData.compactMap { item -> HostInfo? in
                        guard let id = item["id"] as? String,
                              let online = item["online"] as? Bool else { return nil }
                        return HostInfo(id: id, online: online)
                    }
                    self.delegate?.signalingDidReceiveHosts(hosts)
                }

            case "answer":
                let from = json["from"] as? String ?? ""
                if let payload = json["payload"] as? [String: Any] {
                    self.delegate?.signalingDidReceiveAnswer(from: from, payload: payload)
                }

            case "ice-candidate":
                let from = json["from"] as? String ?? ""
                if let payload = json["payload"] as? [String: Any] {
                    self.delegate?.signalingDidReceiveICECandidate(from: from, payload: payload)
                }

            case "host-disconnected":
                if let hostID = json["hostId"] as? String {
                    self.delegate?.signalingDidReceiveHostDisconnected(hostID: hostID)
                }

            case "error":
                let msg = json["message"] as? String ?? "Unknown error"
                logger.error("Signaling error: \(msg)")
                self.delegate?.signalingDidReceiveError(message: msg)

            case "pong":
                break // heartbeat response

            default:
                logger.warning("Unknown message type: \(type)")
            }
        }
    }

    /// 25-second ping timer (matches client.go:pingLoop)
    private func startPingTimer() {
        pingTimer?.invalidate()
        pingTimer = Timer.scheduledTimer(withTimeInterval: 25.0, repeats: true) { [weak self] _ in
            self?.send(SignalingMessage(type: .ping))
        }
    }
}
