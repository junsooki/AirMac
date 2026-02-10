package peer

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"

	"github.com/junsooki/AirMac/internal/signaling"
	"github.com/junsooki/AirMac/internal/transport"
)

// Controller manages the controller side of the WebRTC connection.
type Controller struct {
	pc        *webrtc.PeerConnection
	sig       *signaling.Client
	transport *transport.DataChannelTransport
	hostID    string
}

// NewController creates a Controller peer manager.
func NewController(sig *signaling.Client, hostID string) (*Controller, error) {
	pc, err := NewPeerConnection()
	if err != nil {
		return nil, err
	}

	ctrl := &Controller{
		pc:        pc,
		sig:       sig,
		transport: transport.NewDataChannelTransport(nil, nil),
		hostID:    hostID,
	}

	// Accept data channels from the host.
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("data channel received: %s", dc.Label())
		switch dc.Label() {
		case "frames":
			dc.OnOpen(func() {
				log.Println("frames data channel open")
			})
			ctrl.transport.SetFramesChannel(dc)
		case "input":
			dc.OnOpen(func() {
				log.Println("input data channel open")
			})
			ctrl.transport.SetInputChannel(dc)
		}
	})

	// ICE candidate handling.
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		data, err := json.Marshal(c.ToJSON())
		if err != nil {
			log.Printf("marshal ICE candidate: %v", err)
			return
		}
		_ = sig.SendICECandidate(hostID, data)
	})

	return ctrl, nil
}

// Transport returns the DataChannelTransport.
func (c *Controller) Transport() *transport.DataChannelTransport {
	return c.transport
}

// Connect initiates the WebRTC connection by creating and sending an offer.
func (c *Controller) Connect() error {
	offer, err := c.pc.CreateOffer(nil)
	if err != nil {
		return err
	}

	if err := c.pc.SetLocalDescription(offer); err != nil {
		return err
	}

	offerJSON, err := json.Marshal(offer)
	if err != nil {
		return err
	}

	return c.sig.SendOffer(c.hostID, offerJSON)
}

// HandleAnswer processes an incoming SDP answer.
func (c *Controller) HandleAnswer(payload json.RawMessage) error {
	var answer webrtc.SessionDescription
	if err := json.Unmarshal(payload, &answer); err != nil {
		return err
	}
	return c.pc.SetRemoteDescription(answer)
}

// HandleICECandidate adds a remote ICE candidate.
func (c *Controller) HandleICECandidate(payload json.RawMessage) error {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(payload, &candidate); err != nil {
		return err
	}
	return c.pc.AddICECandidate(candidate)
}

// Close shuts down the peer connection.
func (c *Controller) Close() {
	if c.pc != nil {
		c.pc.Close()
	}
}
