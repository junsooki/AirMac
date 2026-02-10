package peer

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"

	"github.com/junsooki/AirMac/internal/signaling"
	"github.com/junsooki/AirMac/internal/transport"
)

// Host manages the host side of the WebRTC connection.
type Host struct {
	pc        *webrtc.PeerConnection
	sig       *signaling.Client
	transport *transport.DataChannelTransport
	peerID    string // the controller we're connected to
}

// NewHost creates a Host peer manager.
func NewHost(sig *signaling.Client) (*Host, error) {
	pc, err := NewPeerConnection()
	if err != nil {
		return nil, err
	}

	h := &Host{
		pc:  pc,
		sig: sig,
	}

	// Create DataChannels (host is the offerer-side for DCs created here,
	// but we actually answer offers from the controller).
	// We create the channels so they exist before we set local description.
	framesOrdered := false
	framesMaxRetransmits := uint16(0)
	framesDC, err := pc.CreateDataChannel("frames", &webrtc.DataChannelInit{
		Ordered:        &framesOrdered,
		MaxRetransmits: &framesMaxRetransmits,
	})
	if err != nil {
		pc.Close()
		return nil, err
	}

	inputOrdered := true
	inputDC, err := pc.CreateDataChannel("input", &webrtc.DataChannelInit{
		Ordered: &inputOrdered,
	})
	if err != nil {
		pc.Close()
		return nil, err
	}

	h.transport = transport.NewDataChannelTransport(framesDC, inputDC)

	// ICE candidate handling.
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil || h.peerID == "" {
			return
		}
		data, err := json.Marshal(c.ToJSON())
		if err != nil {
			log.Printf("marshal ICE candidate: %v", err)
			return
		}
		_ = sig.SendICECandidate(h.peerID, data)
	})

	return h, nil
}

// Transport returns the DataChannelTransport for sending frames and receiving input.
func (h *Host) Transport() *transport.DataChannelTransport {
	return h.transport
}

// HandleOffer processes an incoming offer from a controller.
func (h *Host) HandleOffer(from string, payload json.RawMessage) error {
	h.peerID = from

	var offer webrtc.SessionDescription
	if err := json.Unmarshal(payload, &offer); err != nil {
		return err
	}

	if err := h.pc.SetRemoteDescription(offer); err != nil {
		return err
	}

	answer, err := h.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	if err := h.pc.SetLocalDescription(answer); err != nil {
		return err
	}

	answerJSON, err := json.Marshal(answer)
	if err != nil {
		return err
	}

	return h.sig.SendAnswer(from, answerJSON)
}

// HandleICECandidate adds a remote ICE candidate.
func (h *Host) HandleICECandidate(payload json.RawMessage) error {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(payload, &candidate); err != nil {
		return err
	}
	return h.pc.AddICECandidate(candidate)
}

// Close shuts down the peer connection.
func (h *Host) Close() {
	if h.pc != nil {
		h.pc.Close()
	}
}
