package decoder

import "image"

// Decoder decodes bytes into an image.
type Decoder interface {
	Decode(data []byte) (*image.RGBA, error)
}
