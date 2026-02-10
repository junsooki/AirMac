package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/junsooki/AirMac/internal/config"
	"github.com/junsooki/AirMac/internal/decoder"
	"github.com/junsooki/AirMac/internal/display"
	"github.com/junsooki/AirMac/internal/peer"
	"github.com/junsooki/AirMac/internal/signaling"
)

func main() {
	cfg := config.ParseControllerFlags()

	if cfg.HostID == "" {
		log.Fatal("Usage: airmac-controller -signaling <url> -host <host-id>")
	}

	log.Printf("AirMac Controller starting")
	log.Printf("  Controller ID: %s", cfg.ControllerID)
	log.Printf("  Signaling:     %s", cfg.SignalingURL)
	log.Printf("  Target host:   %s", cfg.HostID)

	// Decoder.
	dec := decoder.NewJPEGDecoder()

	// Peer manager.
	var ctrlPeer *peer.Controller

	// Display — sends input back to host.
	disp := display.NewEbitenDisplay(func(eventJSON []byte) {
		if ctrlPeer != nil {
			ctrlPeer.Transport().SendInput(eventJSON)
		}
	})

	// Signaling.
	var sig *signaling.Client
	sig = signaling.NewClient(cfg.SignalingURL, cfg.ControllerID, signaling.ClientTypeController, signaling.Handler{
		OnRegistered: func() {
			log.Println("Registered with signaling server")

			// Create peer and send offer.
			var err error
			ctrlPeer, err = peer.NewController(sig, cfg.HostID)
			if err != nil {
				log.Printf("create controller peer: %v", err)
				os.Exit(1)
			}

			// Wire frame receiving.
			ctrlPeer.Transport().OnFrame(func(data []byte) {
				img, err := dec.Decode(data)
				if err != nil {
					return
				}
				disp.SetFrame(img)
			})

			if err := ctrlPeer.Connect(); err != nil {
				log.Printf("controller connect: %v", err)
			}
		},
		OnAnswer: func(from string, payload json.RawMessage) {
			if ctrlPeer != nil {
				if err := ctrlPeer.HandleAnswer(payload); err != nil {
					log.Printf("handle answer: %v", err)
				}
			}
		},
		OnICECandidate: func(from string, payload json.RawMessage) {
			if ctrlPeer != nil {
				if err := ctrlPeer.HandleICECandidate(payload); err != nil {
					log.Printf("handle ICE candidate: %v", err)
				}
			}
		},
		OnError: func(msg string) {
			log.Printf("signaling error: %s", msg)
		},
	})

	if err := sig.Connect(); err != nil {
		log.Fatalf("signaling connect: %v", err)
	}
	defer sig.Close()

	// Ebitengine RunGame must be on the main goroutine (macOS requirement).
	if err := disp.Run(); err != nil {
		log.Fatalf("display: %v", err)
	}

	if ctrlPeer != nil {
		ctrlPeer.Close()
	}
}
