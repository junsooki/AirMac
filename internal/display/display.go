package display

import "image"

// Display renders frames and captures user input.
type Display interface {
	Run() error
}

// InputCallback is called when the user generates an input event.
type InputCallback func(eventJSON []byte)

// FrameSource provides decoded frames to the display.
type FrameSource interface {
	CurrentFrame() *image.RGBA
}
