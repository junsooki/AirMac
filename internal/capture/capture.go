package capture

import (
	"image"
	"time"
)

// Frame represents a captured screen frame.
type Frame struct {
	Image     *image.RGBA
	Timestamp time.Time
}
