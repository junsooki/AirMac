package input

// EventType identifies the kind of input event.
type EventType string

const (
	EventMouseMove    EventType = "mouse_move"
	EventMouseDown    EventType = "mouse_down"
	EventMouseUp      EventType = "mouse_up"
	EventMouseScroll  EventType = "mouse_scroll"
	EventKeyDown      EventType = "key_down"
	EventKeyUp        EventType = "key_up"
)

// MouseButton identifies a mouse button.
type MouseButton int

const (
	MouseButtonLeft   MouseButton = 0
	MouseButtonRight  MouseButton = 1
	MouseButtonMiddle MouseButton = 2
)

// InputEvent is the wire format for input events sent over the data channel.
type InputEvent struct {
	Type    EventType   `json:"type"`
	X       float64     `json:"x,omitempty"`
	Y       float64     `json:"y,omitempty"`
	Button  MouseButton `json:"button,omitempty"`
	KeyCode uint16      `json:"keyCode,omitempty"`
	// Modifier flags (bitfield): 1=Shift, 2=Ctrl, 4=Alt/Option, 8=Cmd
	Modifiers uint8 `json:"modifiers,omitempty"`
	ScrollDX  float64 `json:"scrollDX,omitempty"`
	ScrollDY  float64 `json:"scrollDY,omitempty"`
}
