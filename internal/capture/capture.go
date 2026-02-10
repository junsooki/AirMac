package capture

import (
	"image"
	"time"
)

// Capturer captures screen frames.
type Capturer interface {
	Start() error
	Stop()
	Frames() <-chan *Frame
}

// Frame represents a captured screen frame.
type Frame struct {
	Image     *image.RGBA
	Width     int
	Height    int
	Timestamp time.Time
}
