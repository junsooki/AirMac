package peer

import (
	"log"

	"github.com/pion/webrtc/v4"
)

// ICEServers is the default ICE server configuration.
var ICEServers = []webrtc.ICEServer{
	{URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"}},
}

// NewPeerConnection creates a configured PeerConnection.
func NewPeerConnection() (*webrtc.PeerConnection, error) {
	cfg := webrtc.Configuration{
		ICEServers: ICEServers,
	}
	pc, err := webrtc.NewPeerConnection(cfg)
	if err != nil {
		return nil, err
	}
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("peer connection state: %s", state.String())
	})
	return pc, nil
}
