package transport

// FrameSender sends encoded video frames.
type FrameSender interface {
	SendFrame(data []byte) error
}

// FrameReceiver receives encoded video frames.
type FrameReceiver interface {
	OnFrame(callback func(data []byte))
}

// InputSender sends serialized input events.
type InputSender interface {
	SendInput(data []byte) error
}

// InputReceiver receives serialized input events.
type InputReceiver interface {
	OnInput(callback func(data []byte))
}
