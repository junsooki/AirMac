package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/junsooki/AirMac/internal/capture"
	"github.com/junsooki/AirMac/internal/config"
	"github.com/junsooki/AirMac/internal/encoder"
	"github.com/junsooki/AirMac/internal/input"
	"github.com/junsooki/AirMac/internal/peer"
	"github.com/junsooki/AirMac/internal/permissions"
	"github.com/junsooki/AirMac/internal/signaling"
	"github.com/junsooki/AirMac/internal/transport"
)

func main() {
	cfg := config.ParseHostFlags()

	log.Printf("AirMac Host starting")
	log.Printf("  Host ID:    %s", cfg.HostID)
	log.Printf("  Signaling:  %s", cfg.SignalingURL)
	log.Printf("  Display:    %d", cfg.DisplayIndex)
	log.Printf("  FPS:        %d", cfg.FPS)
	log.Printf("  Quality:    %d", cfg.Quality)

	// Check permissions.
	if !permissions.HasScreenRecording() {
		log.Println("Screen Recording permission not granted. Requesting...")
		permissions.RequestScreenRecording()
		log.Fatal("Please grant Screen Recording permission in System Settings and restart.")
	}
	if !permissions.HasAccessibility() {
		log.Println("Accessibility permission not granted. Requesting...")
		permissions.RequestAccessibility()
		log.Fatal("Please grant Accessibility permission in System Settings and restart.")
	}

	// Screen capture.
	cap, err := capture.NewCGCapturer(cfg.DisplayIndex, cfg.FPS)
	if err != nil {
		log.Fatalf("capture init: %v", err)
	}

	// Encoder.
	enc := encoder.NewJPEGEncoder(cfg.Quality)

	// Input injector.
	injector := input.NewCGEventInjector()

	// Peer manager (created on first offer).
	var hostPeer *peer.Host
	var sig *signaling.Client

	// Signaling.
	sig = signaling.NewClient(cfg.SignalingURL, cfg.HostID, signaling.ClientTypeHost, signaling.Handler{
		OnRegistered: func() {
			log.Println("Registered with signaling server")
		},
		OnOffer: func(from string, payload json.RawMessage) {
			log.Printf("Received offer from %s", from)
			var err error
			if hostPeer != nil {
				hostPeer.Close()
			}
			hostPeer, err = peer.NewHost(sig)
			if err != nil {
				log.Printf("create host peer: %v", err)
				return
			}

			// Wire input receiving.
			hostPeer.Transport().OnInput(func(data []byte) {
				var evt input.InputEvent
				if err := json.Unmarshal(data, &evt); err != nil {
					log.Printf("unmarshal input: %v", err)
					return
				}
				injector.Inject(&evt)
			})

			if err := hostPeer.HandleOffer(from, payload); err != nil {
				log.Printf("handle offer: %v", err)
				return
			}

			go streamFrames(cap.Frames(), enc, hostPeer.Transport())
		},
		OnICECandidate: func(from string, payload json.RawMessage) {
			if hostPeer != nil {
				if err := hostPeer.HandleICECandidate(payload); err != nil {
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

	if err := cap.Start(); err != nil {
		log.Fatalf("capture start: %v", err)
	}
	defer cap.Stop()

	log.Printf("Host ready. Share this ID with controllers: %s", cfg.HostID)

	// Wait for interrupt.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	if hostPeer != nil {
		hostPeer.Close()
	}
}

func streamFrames(frames <-chan *capture.Frame, enc *encoder.JPEGEncoder, t *transport.DataChannelTransport) {
	for frame := range frames {
		data, err := enc.Encode(frame.Image)
		if err != nil {
			log.Printf("encode frame: %v", err)
			continue
		}
		if err := t.SendFrame(data); err != nil {
			continue
		}
	}
}
