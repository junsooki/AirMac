package decoder

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
)

// JPEGDecoder decodes JPEG bytes into *image.RGBA.
type JPEGDecoder struct{}

func NewJPEGDecoder() *JPEGDecoder {
	return &JPEGDecoder{}
}

func (d *JPEGDecoder) Decode(data []byte) (*image.RGBA, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	// Convert to RGBA if needed.
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, nil
	}
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, b.Min, draw.Src)
	return rgba, nil
}
