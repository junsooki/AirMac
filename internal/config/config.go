package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
)

// Config holds all runtime configuration.
type Config struct {
	SignalingURL string
	HostID       string
	DisplayIndex int
	FPS          int
	Quality      int
}

// ParseHostFlags parses flags for the host binary.
func ParseHostFlags() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.SignalingURL, "signaling", "ws://localhost:8080", "Signaling server WebSocket URL")
	flag.StringVar(&cfg.HostID, "id", "", "Host ID (auto-generated if empty)")
	flag.IntVar(&cfg.DisplayIndex, "display", 0, "Display index to capture (0 = primary)")
	flag.IntVar(&cfg.FPS, "fps", 30, "Target frames per second")
	flag.IntVar(&cfg.Quality, "quality", 70, "JPEG quality (1-100)")
	flag.Parse()

	if cfg.HostID == "" {
		cfg.HostID = fmt.Sprintf("host-%s", randomID())
	}
	return cfg
}

// ControllerConfig holds configuration for the controller binary.
type ControllerConfig struct {
	SignalingURL string
	ControllerID string
	HostID       string
}

// ParseControllerFlags parses flags for the controller binary.
func ParseControllerFlags() *ControllerConfig {
	cfg := &ControllerConfig{}
	flag.StringVar(&cfg.SignalingURL, "signaling", "ws://localhost:8080", "Signaling server WebSocket URL")
	flag.StringVar(&cfg.ControllerID, "id", "", "Controller ID (auto-generated if empty)")
	flag.StringVar(&cfg.HostID, "host", "", "Host ID to connect to (required)")
	flag.Parse()

	if cfg.ControllerID == "" {
		cfg.ControllerID = fmt.Sprintf("controller-%s", randomID())
	}
	return cfg
}

func randomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
