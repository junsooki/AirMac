package signaling

import "encoding/json"

// Message types for signaling protocol.
const (
	TypeRegister     = "register"
	TypeRegistered   = "registered"
	TypeListHosts    = "list-hosts"
	TypeHosts        = "hosts"
	TypeHostsUpdated = "hosts-updated"
	TypeOffer        = "offer"
	TypeAnswer       = "answer"
	TypeICECandidate = "ice-candidate"
	TypePing         = "ping"
	TypePong         = "pong"
	TypeError        = "error"
	TypeHostDisconnected = "host-disconnected"
)

// ClientType distinguishes host from controller.
const (
	ClientTypeHost       = "host"
	ClientTypeController = "controller"
)

// Message is the envelope for all signaling messages.
type Message struct {
	Type       string          `json:"type"`
	ID         string          `json:"id,omitempty"`
	ClientType string          `json:"clientType,omitempty"`
	From       string          `json:"from,omitempty"`
	Target     string          `json:"target,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	List       []HostInfo      `json:"list,omitempty"`
	HostID     string          `json:"hostId,omitempty"`
	Msg        string          `json:"message,omitempty"`
	Timestamp  int64           `json:"timestamp,omitempty"`
}

// HostInfo describes a host in the host list.
type HostInfo struct {
	ID     string `json:"id"`
	Online bool   `json:"online"`
}
