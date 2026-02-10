package encoder

import (
	"bytes"
	"image"
	"image/jpeg"
)

// JPEGEncoder encodes frames as JPEG.
type JPEGEncoder struct {
	quality int
}

// NewJPEGEncoder creates a JPEG encoder with the given quality (1-100).
func NewJPEGEncoder(quality int) *JPEGEncoder {
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}
	return &JPEGEncoder{quality: quality}
}

func (e *JPEGEncoder) Encode(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256 * 1024) // pre-allocate 256KB
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: e.quality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

