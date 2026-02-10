package encoder

import "image"

// Encoder encodes an image into bytes.
type Encoder interface {
	Encode(img *image.RGBA) ([]byte, error)
	SetQuality(quality int)
}
