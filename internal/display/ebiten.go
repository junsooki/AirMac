package display

import (
	"encoding/json"
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/junsooki/AirMac/internal/input"
)

// InputCallback is called when the user generates an input event.
type InputCallback func(eventJSON []byte)

// EbitenDisplay renders the remote screen using Ebitengine and captures input.
type EbitenDisplay struct {
	mu          sync.Mutex
	frame       *image.RGBA
	ebitenImage *ebiten.Image
	onInput     InputCallback

	screenW int
	screenH int

	prevMouseX int
	prevMouseY int
}

// NewEbitenDisplay creates an Ebitengine-based display.
func NewEbitenDisplay(onInput InputCallback) *EbitenDisplay {
	return &EbitenDisplay{
		onInput: onInput,
		screenW: 1280,
		screenH: 720,
	}
}

// SetFrame updates the displayed frame (called from network goroutine).
func (d *EbitenDisplay) SetFrame(img *image.RGBA) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frame = img
	if img != nil {
		w := img.Bounds().Dx()
		h := img.Bounds().Dy()
		if w != d.screenW || h != d.screenH {
			d.screenW = w
			d.screenH = h
		}
	}
}

// Run starts the Ebitengine game loop. Must be called from the main goroutine.
func (d *EbitenDisplay) Run() error {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("AirMac Controller")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(d)
}

// --- ebiten.Game interface ---

func (d *EbitenDisplay) Update() error {
	d.captureMouseInput()
	d.captureKeyboardInput()
	return nil
}

func (d *EbitenDisplay) Draw(screen *ebiten.Image) {
	d.mu.Lock()
	frame := d.frame
	d.mu.Unlock()

	if frame == nil {
		return
	}

	if d.ebitenImage == nil ||
		d.ebitenImage.Bounds().Dx() != frame.Bounds().Dx() ||
		d.ebitenImage.Bounds().Dy() != frame.Bounds().Dy() {
		d.ebitenImage = ebiten.NewImage(frame.Bounds().Dx(), frame.Bounds().Dy())
	}
	d.ebitenImage.WritePixels(frame.Pix)

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	fw, fh := float64(frame.Bounds().Dx()), float64(frame.Bounds().Dy())
	scale, offsetX, offsetY := aspectFitTransform(float64(sw), float64(sh), fw, fh)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(d.ebitenImage, op)
}

func (d *EbitenDisplay) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// --- Input capture ---

func (d *EbitenDisplay) captureMouseInput() {
	mx, my := ebiten.CursorPosition()

	sw, sh := ebiten.WindowSize()
	d.mu.Lock()
	frame := d.frame
	d.mu.Unlock()
	if frame == nil {
		return
	}

	fw := float64(frame.Bounds().Dx())
	fh := float64(frame.Bounds().Dy())
	scale, offsetX, offsetY := aspectFitTransform(float64(sw), float64(sh), fw, fh)
	remoteX := (float64(mx) - offsetX) / scale
	remoteY := (float64(my) - offsetY) / scale

	// Mouse move.
	if mx != d.prevMouseX || my != d.prevMouseY {
		d.prevMouseX = mx
		d.prevMouseY = my
		d.sendInput(input.InputEvent{
			Type: input.EventMouseMove,
			X:    remoteX,
			Y:    remoteY,
		})
	}

	// Mouse buttons.
	buttons := []struct {
		eb  ebiten.MouseButton
		btn input.MouseButton
	}{
		{ebiten.MouseButtonLeft, input.MouseButtonLeft},
		{ebiten.MouseButtonRight, input.MouseButtonRight},
		{ebiten.MouseButtonMiddle, input.MouseButtonMiddle},
	}
	for _, b := range buttons {
		if inpututil.IsMouseButtonJustPressed(b.eb) {
			d.sendInput(input.InputEvent{Type: input.EventMouseDown, X: remoteX, Y: remoteY, Button: b.btn})
		}
		if inpututil.IsMouseButtonJustReleased(b.eb) {
			d.sendInput(input.InputEvent{Type: input.EventMouseUp, X: remoteX, Y: remoteY, Button: b.btn})
		}
	}

	// Scroll.
	_, scrollY := ebiten.Wheel()
	if scrollY != 0 {
		d.sendInput(input.InputEvent{Type: input.EventMouseScroll, ScrollDY: scrollY})
	}
}

func (d *EbitenDisplay) captureKeyboardInput() {
	// Check all keys.
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if inpututil.IsKeyJustPressed(k) {
			d.sendInput(input.InputEvent{
				Type:      input.EventKeyDown,
				KeyCode:   ebitenKeyToMacKeyCode(k),
				Modifiers: currentModifiers(),
			})
		}
		if inpututil.IsKeyJustReleased(k) {
			d.sendInput(input.InputEvent{
				Type:      input.EventKeyUp,
				KeyCode:   ebitenKeyToMacKeyCode(k),
				Modifiers: currentModifiers(),
			})
		}
	}
}

func (d *EbitenDisplay) sendInput(e input.InputEvent) {
	if d.onInput == nil {
		return
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	d.onInput(data)
}

func currentModifiers() uint8 {
	var m uint8
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		m |= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		m |= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		m |= 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyMeta) {
		m |= 8
	}
	return m
}

// aspectFitTransform returns scale and offsets to fit frame into view with letterboxing.
func aspectFitTransform(viewW, viewH, frameW, frameH float64) (scale, offsetX, offsetY float64) {
	scale = math.Min(viewW/frameW, viewH/frameH)
	offsetX = (viewW - frameW*scale) / 2
	offsetY = (viewH - frameH*scale) / 2
	return
}

// ebitenKeyToMacKeyCode maps Ebitengine key codes to macOS virtual key codes.
func ebitenKeyToMacKeyCode(k ebiten.Key) uint16 {
	m := map[ebiten.Key]uint16{
		ebiten.KeyA: 0x00, ebiten.KeyS: 0x01, ebiten.KeyD: 0x02, ebiten.KeyF: 0x03,
		ebiten.KeyH: 0x04, ebiten.KeyG: 0x05, ebiten.KeyZ: 0x06, ebiten.KeyX: 0x07,
		ebiten.KeyC: 0x08, ebiten.KeyV: 0x09, ebiten.KeyB: 0x0B, ebiten.KeyQ: 0x0C,
		ebiten.KeyW: 0x0D, ebiten.KeyE: 0x0E, ebiten.KeyR: 0x0F, ebiten.KeyY: 0x10,
		ebiten.KeyT: 0x11, ebiten.Key1: 0x12, ebiten.Key2: 0x13, ebiten.Key3: 0x14,
		ebiten.Key4: 0x15, ebiten.Key6: 0x16, ebiten.Key5: 0x17, ebiten.Key9: 0x19,
		ebiten.Key7: 0x1A, ebiten.Key8: 0x1C, ebiten.Key0: 0x1D, ebiten.KeyO: 0x1F,
		ebiten.KeyU: 0x20, ebiten.KeyI: 0x22, ebiten.KeyP: 0x23, ebiten.KeyL: 0x25,
		ebiten.KeyJ: 0x26, ebiten.KeyK: 0x28, ebiten.KeyN: 0x2D, ebiten.KeyM: 0x2E,
		ebiten.KeyEnter: 0x24, ebiten.KeyTab: 0x30, ebiten.KeySpace: 0x31,
		ebiten.KeyBackspace: 0x33, ebiten.KeyEscape: 0x35,
		ebiten.KeyArrowLeft: 0x7B, ebiten.KeyArrowRight: 0x7C,
		ebiten.KeyArrowDown: 0x7D, ebiten.KeyArrowUp: 0x7E,
		ebiten.KeyF1: 0x7A, ebiten.KeyF2: 0x78, ebiten.KeyF3: 0x63, ebiten.KeyF4: 0x76,
		ebiten.KeyF5: 0x60, ebiten.KeyF6: 0x61, ebiten.KeyF7: 0x62, ebiten.KeyF8: 0x64,
		ebiten.KeyF9: 0x65, ebiten.KeyF10: 0x6D, ebiten.KeyF11: 0x67, ebiten.KeyF12: 0x6F,
		ebiten.KeyDelete: 0x75, ebiten.KeyHome: 0x73, ebiten.KeyEnd: 0x77,
		ebiten.KeyPageUp: 0x74, ebiten.KeyPageDown: 0x79,
	}
	if code, ok := m[k]; ok {
		return code
	}
	return 0xFF // unmapped
}
