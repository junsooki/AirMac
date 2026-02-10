import Foundation

// MARK: - Message types (must match internal/signaling/messages.go)

enum SignalingMessageType: String, Codable {
    case register = "register"
    case registered = "registered"
    case listHosts = "list-hosts"
    case hosts = "hosts"
    case hostsUpdated = "hosts-updated"
    case offer = "offer"
    case answer = "answer"
    case iceCandidate = "ice-candidate"
    case ping = "ping"
    case pong = "pong"
    case error = "error"
    case hostDisconnected = "host-disconnected"
}

enum ClientType: String, Codable {
    case host = "host"
    case controller = "controller"
}

// MARK: - Wire format (must match messages.go Message struct)

struct SignalingMessage: Codable {
    let type: SignalingMessageType
    var id: String?
    var clientType: ClientType?
    var from: String?
    var target: String?
    var payload: AnyCodable?
    var list: [HostInfo]?
    var hostId: String?
    var message: String?
    var timestamp: Int64?

    // Custom encoding to match Go's omitempty behavior
    enum CodingKeys: String, CodingKey {
        case type, id, clientType, from, target, payload, list, hostId, message, timestamp
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(type, forKey: .type)
        try container.encodeIfPresent(id, forKey: .id)
        try container.encodeIfPresent(clientType, forKey: .clientType)
        try container.encodeIfPresent(from, forKey: .from)
        try container.encodeIfPresent(target, forKey: .target)
        try container.encodeIfPresent(payload, forKey: .payload)
        try container.encodeIfPresent(list, forKey: .list)
        try container.encodeIfPresent(hostId, forKey: .hostId)
        try container.encodeIfPresent(message, forKey: .message)
        try container.encodeIfPresent(timestamp, forKey: .timestamp)
    }
}

struct HostInfo: Codable, Identifiable, Hashable {
    let id: String
    let online: Bool
}

// MARK: - AnyCodable wrapper for payload (preserves raw JSON)

struct AnyCodable: Codable {
    let value: Any

    init(_ value: Any) {
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict.mapValues { $0.value }
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map { $0.value }
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else {
            value = NSNull()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch value {
        case let dict as [String: Any]:
            try container.encode(dict.mapValues { AnyCodable($0) })
        case let array as [Any]:
            try container.encode(array.map { AnyCodable($0) })
        case let string as String:
            try container.encode(string)
        case let int as Int:
            try container.encode(int)
        case let double as Double:
            try container.encode(double)
        case let bool as Bool:
            try container.encode(bool)
        default:
            try container.encodeNil()
        }
    }
}
