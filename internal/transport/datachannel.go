package transport

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

// DataChannelTransport implements frame and input transport over WebRTC DataChannels.
type DataChannelTransport struct {
	framesDC *webrtc.DataChannel
	inputDC  *webrtc.DataChannel

	onFrame func(data []byte)
	onInput func(data []byte)
}

// NewDataChannelTransport wraps two DataChannels (frames + input).
func NewDataChannelTransport(framesDC, inputDC *webrtc.DataChannel) *DataChannelTransport {
	t := &DataChannelTransport{
		framesDC: framesDC,
		inputDC:  inputDC,
	}

	if framesDC != nil {
		framesDC.OnMessage(func(msg webrtc.DataChannelMessage) {
			if t.onFrame != nil {
				t.onFrame(msg.Data)
			}
		})
	}

	if inputDC != nil {
		inputDC.OnMessage(func(msg webrtc.DataChannelMessage) {
			if t.onInput != nil {
				t.onInput(msg.Data)
			}
		})
	}

	return t
}

func (t *DataChannelTransport) SendFrame(data []byte) error {
	if t.framesDC == nil {
		return fmt.Errorf("frames data channel not set")
	}
	return t.framesDC.Send(data)
}

func (t *DataChannelTransport) SendInput(data []byte) error {
	if t.inputDC == nil {
		return fmt.Errorf("input data channel not set")
	}
	return t.inputDC.Send(data)
}

func (t *DataChannelTransport) OnFrame(cb func(data []byte)) {
	t.onFrame = cb
}

func (t *DataChannelTransport) OnInput(cb func(data []byte)) {
	t.onInput = cb
}

// SetFramesChannel sets or replaces the frames DataChannel (used when receiving negotiated channels).
func (t *DataChannelTransport) SetFramesChannel(dc *webrtc.DataChannel) {
	t.framesDC = dc
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if t.onFrame != nil {
			t.onFrame(msg.Data)
		}
	})
}

// SetInputChannel sets or replaces the input DataChannel.
func (t *DataChannelTransport) SetInputChannel(dc *webrtc.DataChannel) {
	t.inputDC = dc
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if t.onInput != nil {
			t.onInput(msg.Data)
		}
	})
}
